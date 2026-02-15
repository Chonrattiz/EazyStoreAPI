package controllers

import (
	"EazyStoreAPI/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateCreditSale(c *gin.Context, db *gorm.DB) {
	var input models.Sale // ใช้ Struct ที่เราคุยกันก่อนหน้า

	// 1. รับข้อมูล JSON จาก Flutter
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ข้อมูลไม่ถูกต้อง: " + err.Error()})
		return
	}

	// ตรวจสอบเบื้องต้น: ถ้าเป็นการค้างชำระ ต้องมี debtor_id
	if input.PaymentMethod == "credit" && input.DebtorID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "การขายแบบค้างชำระต้องระบุลูกหนี้"})
		return
	}

	// 2. เริ่ม Database Transaction
	err := db.Transaction(func(tx *gorm.DB) error {

		// ก. บันทึกหัวบิล (sales)
		if err := tx.Create(&input).Error; err != nil {
			return err
		}

		// ข. บันทึกรายการสินค้า (sale_items)
		// (ถ้าใน Struct Sale มีฟิลด์ SaleItems ระบบจะบันทึกให้อัตโนมัติใน tx.Create ด้านบน)

		// ค. อัปเดตยอดหนี้ในตาราง debtors
		if input.PaymentMethod == "credit" {
			// คำนวณยอดที่ค้างจริง (ราคาสุทธิ - ยอดที่จ่ายมาบางส่วน)
			debtAmount := input.NetPrice - input.Pay

			// SQL: UPDATE debtors SET total_debt = total_debt + ? WHERE debtor_id = ?
			result := tx.Table("debtors").
				Where("debtor_id = ?", input.DebtorID).
				UpdateColumn("total_debt", gorm.Expr("total_debt + ?", debtAmount))

			if result.Error != nil {
				return result.Error
			}
		}

		return nil
	})

	// 3. ส่งคำตอบกลับ
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "บันทึกไม่สำเร็จ: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "บันทึกการขายแบบค้างชำระสำเร็จ", "sale_id": input.SaleID})
}
