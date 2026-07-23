package controllers

import (
	"EazyStoreAPI/database"
	"EazyStoreAPI/models"
	"net/http"

	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateCreditSale godoc
// @Summary      เพิ่มรายการขายค้างชำระ (เฉพาะ Credit เท่านั้น)
// @Description  สร้างบิลขายและอัปเดตยอดหนี้คงค้างของลูกหนี้ในคราวเดียว
// @Tags         Sale
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Sale body models.Sale true "ข้อมูลรายการขายค้างชำระ"
// @Success      200  {object} map[string]interface{}
// @Failure      400  {object} map[string]string
// @Router       /api/createCreditSale [post]
func CreateCreditSale(c *gin.Context) {
	var input models.Sale

	// 1. รับข้อมูล JSON
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON: " + err.Error()})
		return
	}

	// 2. Validation เบื้องต้น
	if input.PaymentMethod != "ค้างชำระ" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "เส้นทางนี้สำหรับรายการค้างชำระเท่านั้น"})
		return
	}

	if input.DebtorID == nil || *input.DebtorID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาระบุรหัสลูกหนี้"})
		return
	}

	// 3. เริ่ม Transaction
	err := database.DB.Transaction(func(tx *gorm.DB) error {

		// ก. ตรวจสอบว่าลูกหนี้เป็นของร้านนี้จริงไหม และดึงข้อมูลมาเช็ควงเงิน
		var debtor models.Debtor
		if err := tx.Where("debtor_id = ? AND shop_id = ?", input.DebtorID, input.ShopID).First(&debtor).Error; err != nil {
			return errors.New("ไม่พบข้อมูลลูกหนี้ในร้านค้าของคุณ หรือลูกหนี้ไม่มีสิทธิ์ค้างชำระ")
		}

		// ข. เช็ควงเงินหนี้ (Credit Limit)
		amountToCharge := input.NetPrice - input.Pay
		if (debtor.CurrentDebt + amountToCharge) > debtor.CreditLimit {
			return fmt.Errorf("ยอดหนี้เกินวงเงินที่กำหนด (คงเหลือที่ค้างได้: %.2f บาท)", debtor.CreditLimit-debtor.CurrentDebt)
		}

		// ค. ตรวจสอบสินค้า เช็คสต๊อก และทำการตัดสต๊อก (รองรับขายเป็นหน่วยขายเพิ่มเติม เช่น ลัง)
		for i := range input.SaleItems {
			item := &input.SaleItems[i]

			var product models.Product
			// เช็คว่าสินค้ามีอยู่จริงและเป็นของร้านนี้
			if err := tx.Where("product_id = ? AND shop_id = ?", item.ProductID, input.ShopID).First(&product).Error; err != nil {
				return fmt.Errorf("ไม่พบสินค้ารหัส %d หรือไม่ใช่สินค้าของร้านคุณ", item.ProductID)
			}

			// เซิร์ฟเวอร์เป็นคนแปลงหน่วย/ตัดสต๊อกเองเสมอ ไม่เชื่อ conversion_qty ที่ client ส่งมา
			conv := 1
			var unitName *string
			if item.ProductUnitID != nil {
				var unit models.ProductUnit
				if err := tx.Where("product_unit_id = ? AND product_id = ?", *item.ProductUnitID, item.ProductID).
					First(&unit).Error; err != nil {
					return fmt.Errorf("ไม่พบหน่วยขายที่เลือกของสินค้า '%s'", product.Name)
				}
				conv = unit.ConversionQty
				unitName = &unit.UnitName
			}
			item.ConversionQty = conv
			item.UnitName = unitName

			needed := item.Amount * conv

			// เช็คว่าสต๊อกพอขายหรือไม่ (นับเป็นหน่วยฐานเสมอ)
			if product.Stock < needed {
				if conv > 1 {
					return fmt.Errorf("สินค้า '%s' มีสต๊อกไม่พอ (ต้องการ %d %s = %d %s, คงเหลือ %d %s)",
						product.Name, item.Amount, *unitName, needed, product.Unit, product.Stock, product.Unit)
				}
				return fmt.Errorf("สินค้า '%s' มีสต๊อกไม่พอ (คงเหลือ %d %s, ต้องการ %d %s)",
					product.Name, product.Stock, product.Unit, item.Amount, product.Unit)
			}

			// ทำการตัดสต๊อก
			if err := tx.Model(&models.Product{}).
				Where("product_id = ?", item.ProductID).
				UpdateColumn("stock", gorm.Expr("stock - ?", needed)).Error; err != nil {
				return fmt.Errorf("ไม่สามารถตัดสต๊อกสินค้า '%s' ได้: %v", product.Name, err)
			}
		}

		// ง. บันทึกข้อมูลการขาย (Sales & SaleItems)
		// หมายเหตุ: GORM จะบันทึก SaleItems ให้โดยอัตโนมัติถ้า Struct Sale มี SaleItems
		if err := tx.Create(&input).Error; err != nil {
			return err
		}

		// จ. อัปเดตยอดหนี้ปัจจุบัน (current_debt)
		if err := tx.Table("debtors").
			Where("debtor_id = ?", input.DebtorID).
			UpdateColumn("current_debt", gorm.Expr("current_debt + ?", amountToCharge)).
			Error; err != nil {
			return err
		}

		return nil
	})

	// 4. จัดการผลลัพธ์
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "บันทึกไม่สำเร็จ: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "บันทึกรายการค้างชำระและอัปเดตยอดหนี้สำเร็จ",
		"sale_id": input.SaleID,
	})
}
