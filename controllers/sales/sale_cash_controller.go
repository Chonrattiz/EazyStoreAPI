package controllers

import (
	"EazyStoreAPI/database"
	"EazyStoreAPI/models"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateSale ฟังก์ชันบันทึกการขาย รองรับทั้ง จ่ายเงินสด และ โอนจ่าย
func CreateSale(c *gin.Context) {
	// โครงสร้างรับข้อมูลจาก Frontend
	var input struct {
		ShopID        int       `json:"shop_id" binding:"required"`
		DebtorID      *int      `json:"debtor_id"`      // กรณีจ่ายสด/โอนจ่าย ค่านี้จะเป็น null
		NetPrice      float64   `json:"net_price" binding:"required"`
		Pay           float64   `json:"pay" binding:"required"`
		PaymentMethod string    `json:"payment_method" binding:"required"` // รับ "จ่ายเงินสด" หรือ "โอนจ่าย"
		Note          *string   `json:"note"`           // ใช้ Pointer เพื่อให้รองรับค่า null จาก JSON
		CreatedBuy    string    `json:"created_buy" binding:"required"`
		SaleItems     []struct {
			ProductID     int     `json:"product_id" binding:"required"`
			Amount        int     `json:"amount" binding:"required"`
			PricePerUnit  float64 `json:"price_per_unit" binding:"required"`
			TotalPrice    float64 `json:"total_price" binding:"required"`
			ProductUnitID *int    `json:"product_unit_id"`
		} `json:"sale_items" binding:"required"`
	}

	// 1. Bind JSON และตรวจสอบข้อมูลพื้นฐาน
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 2. เริ่มต้น Transaction
	tx := database.DB.Begin()

	// 3. บันทึกลงตาราง sales
	sale := models.Sale{
		ShopID:        input.ShopID,
		DebtorID:      input.DebtorID,
		NetPrice:      input.NetPrice,
		Pay:           input.Pay,
		PaymentMethod: input.PaymentMethod, // ใช้ค่าที่ส่งมาจาก Flutter (จ่ายเงินสด/โอนจ่าย)
		Note:          "",
		CreatedAt:     time.Now(),
		CreatedBuy:    input.CreatedBuy,
	}
    
	// จัดการเรื่อง Note ถ้าส่งมาเป็น null
	if input.Note != nil {
		sale.Note = *input.Note
	}

	if err := tx.Create(&sale).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create sale record"})
		return
	}

	// 4. วนลูปบันทึกลงตาราง sale_items (รองรับขายเป็นหน่วยขายเพิ่มเติม เช่น ลัง)
	for _, item := range input.SaleItems {
		var product models.Product
		// เช็คว่าสินค้ามีอยู่จริงและเป็นของร้านนี้
		if err := tx.Where("product_id = ? AND shop_id = ?", item.ProductID, input.ShopID).First(&product).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("ไม่พบสินค้ารหัส %d หรือไม่ใช่สินค้าของร้านคุณ", item.ProductID)})
			return
		}

		// เซิร์ฟเวอร์เป็นคนแปลงหน่วย/ตัดสต๊อกเองเสมอ ไม่เชื่อ conversion_qty ที่ client ส่งมา
		conv := 1
		var unitName *string
		if item.ProductUnitID != nil {
			var unit models.ProductUnit
			if err := tx.Where("product_unit_id = ? AND product_id = ?", *item.ProductUnitID, item.ProductID).
				First(&unit).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("ไม่พบหน่วยขายที่เลือกของสินค้า '%s'", product.Name)})
				return
			}
			conv = unit.ConversionQty
			unitName = &unit.UnitName
		}

		needed := item.Amount * conv

		// เช็คสต๊อกก่อนตัด (เดิม path นี้ไม่เคยเช็คมาก่อน ปล่อยให้ติดลบได้ — เพิ่มเช็คตอนนี้
		// เพราะขายเป็นลังแล้วสต๊อกไม่พอจะทำให้สต๊อกติดลบเยอะโดยไม่รู้ตัว)
		if product.Stock < needed {
			tx.Rollback()
			if conv > 1 {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf(
					"สินค้า '%s' มีสต๊อกไม่พอ (ต้องการ %d %s = %d %s, คงเหลือ %d %s)",
					product.Name, item.Amount, *unitName, needed, product.Unit, product.Stock, product.Unit)})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf(
				"สินค้า '%s' มีสต๊อกไม่พอ (คงเหลือ %d %s, ต้องการ %d %s)",
				product.Name, product.Stock, product.Unit, item.Amount, product.Unit)})
			return
		}

		saleItem := models.SaleItem{
			SaleID:        sale.SaleID, // ID จากที่เพิ่ง Save เมื่อครู่
			ProductID:     item.ProductID,
			Amount:        item.Amount,
			PricePerUnit:  item.PricePerUnit,
			TotalPrice:    item.TotalPrice,
			ProductUnitID: item.ProductUnitID,
			UnitName:      unitName,
			ConversionQty: conv,
		}

		if err := tx.Create(&saleItem).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record item: " + err.Error()})
			return
		}

		// 5. ตัดสต็อกสินค้า
		if err := tx.Model(&models.Product{}).Where("product_id = ?", item.ProductID).
			UpdateColumn("stock", gorm.Expr("stock - ?", needed)).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update stock"})
			return
		}
	}

	// 6. ยืนยันการบันทึกทั้งหมด
	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message": "บันทึกการขายสำเร็จ",
		"sale_id": sale.SaleID,
	})
}
