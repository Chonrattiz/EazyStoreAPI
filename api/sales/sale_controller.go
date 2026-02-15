package controllers

import (
	"EazyStoreAPI/database"
	"EazyStoreAPI/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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
			ProductID    int     `json:"product_id" binding:"required"`
			Amount       int     `json:"amount" binding:"required"`
			PricePerUnit float64 `json:"price_per_unit" binding:"required"`
			TotalPrice   float64 `json:"total_price" binding:"required"`
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

	// 4. วนลูปบันทึกลงตาราง sale_items
	for _, item := range input.SaleItems {
		saleItem := models.SaleItem{
			SaleID:       sale.SaleID, // ID จากที่เพิ่ง Save เมื่อครู่
			ProductID:    item.ProductID,
			Amount:       item.Amount,
			PricePerUnit: item.PricePerUnit,
			TotalPrice:   item.TotalPrice,
		}

		if err := tx.Create(&saleItem).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record item: " + err.Error()})
			return
		}

		// 5. ตัดสต็อกสินค้า (Optional แนะนำให้ทำ)
		if err := tx.Table("products").Where("product_id = ?", item.ProductID).
			UpdateColumn("stock", database.DB.Raw("stock - ?", item.Amount)).Error; err != nil {
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