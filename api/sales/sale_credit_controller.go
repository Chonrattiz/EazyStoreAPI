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

		// ค. ตรวจสอบสินค้าทุกรายการว่าเป็นของร้านนี้จริงไหม (กันพลาด)
		for _, item := range input.SaleItems {
			var productCount int64
			tx.Table("products").Where("product_id = ? AND shop_id = ?", item.ProductID, input.ShopID).Count(&productCount)
			if productCount == 0 {
				return fmt.Errorf("สินค้ารหัส %d ไม่ใช่ของร้านคุณ", item.ProductID)
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
