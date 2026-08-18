package controllers

import (
	"EazyStoreAPI/database"
	"EazyStoreAPI/middleware"
	"EazyStoreAPI/models"
	"net/http"
	"time"

	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateCreditSale(c *gin.Context) {
	var input models.Sale

	// 1. รับข้อมูล JSON
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ข้อมูลที่ส่งมาไม่ถูกต้องหรือไม่ครบถ้วน"})
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

	if input.Pay < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "จำนวนเงินที่จ่ายต้องไม่ติดลบ"})
		return
	}

	// จ่ายครบ/จ่ายเกิน = ไม่มียอดค้าง การชำระ → ต้องบันทึกเป็นขายเงินสดแทน
	if input.Pay >= input.NetPrice {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "รายการนี้จ่ายครบแล้ว ไม่มียอดค้างชำระ กรุณาบันทึกเป็นการขายเงินสดแทน",
		})
		return
	}

	// 2.5 shop_id มาจาก body ต้องตรวจว่าเป็นร้านของ user ที่ล็อกอินอยู่จริง
	if !middleware.RequireShopAccess(c, input.ShopID) {
		return
	}

	// 3. เริ่ม Transaction
	tx := database.DB.Begin()

	//  ตรวจสอบว่าลูกหนี้เป็นของร้านนี้จริงไหม และดึงข้อมูลมาเช็ควงเงิน
	var debtor models.Debtor
	if err := tx.Where("debtor_id = ? AND shop_id = ?", input.DebtorID, input.ShopID).First(&debtor).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "ไม่พบข้อมูลลูกหนี้ในร้านค้าของคุณ หรือลูกหนี้ไม่มีสิทธิ์ค้างชำระ"})
		return
	}

	//  เช็ควงเงินหนี้ (Credit Limit)
	amountToCharge := input.NetPrice - input.Pay
	if (debtor.CurrentDebt + amountToCharge) > debtor.CreditLimit {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("ยอดหนี้เกินวงเงินที่กำหนด (คงเหลือที่ค้างได้: %.2f บาท)", debtor.CreditLimit-debtor.CurrentDebt)})
		return
	}

	//  ตรวจสอบสินค้า เช็คสต๊อก และทำการตัดสต๊อก
	for _, item := range input.SaleItems {
		var product models.Product
		// เช็คว่าสินค้ามีอยู่จริงและเป็นของร้านนี้
		if err := tx.Where("product_id = ? AND shop_id = ?", item.ProductID, input.ShopID).First(&product).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("ไม่พบสินค้ารหัส %d หรือไม่ใช่สินค้าของร้านคุณ", item.ProductID)})
			return
		}

		// เช็คว่าสต๊อกพอขายหรือไม่
		if product.Stock < item.Amount {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("สินค้า '%s' มีสต๊อกไม่พอ (คงเหลือ %d ชิ้น, ต้องการ %d ชิ้น)", product.Name, product.Stock, item.Amount)})
			return
		}

		// ทำการตัดสต๊อก
		if err := tx.Model(&models.Product{}).
			Where("product_id = ? AND shop_id = ?", item.ProductID, input.ShopID).
			UpdateColumn("stock", gorm.Expr("stock - ?", item.Amount)).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("ไม่สามารถตัดสต๊อกสินค้า '%s' ได้", product.Name)})
			return
		}
	}

	//  บันทึกข้อมูลการขาย (Sales & SaleItems)
	// หมายเหตุ: GORM จะบันทึก SaleItems ให้โดยอัตโนมัติถ้า Struct Sale มี SaleItems
	// เซ็ตวันที่/เวลาเป็นเวลาไทย (Asia/Bangkok) เพราะ Flutter ไม่ได้ส่ง created_time มา
	// และเซิร์ฟเวอร์มักตั้ง timezone เป็น UTC จะทำให้เพี้ยนไป 7 ชั่วโมง
	loc, errLoc := time.LoadLocation("Asia/Bangkok")
	if errLoc != nil {
		loc = time.FixedZone("ICT", 7*60*60)
	}
	now := time.Now().In(loc)
	nowTime := now.Format("15:04:05")
	input.CreatedAt = now
	input.CreatedTime = &nowTime

	if err := tx.Create(&input).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถบันทึกรายการขายได้"})
		return
	}

	// อัปเดตยอดหนี้ปัจจุบัน (current_debt)
	if err := tx.Table("debtors").
		Where("debtor_id = ? AND shop_id = ?", input.DebtorID, input.ShopID).
		UpdateColumn("current_debt", gorm.Expr("current_debt + ?", amountToCharge)).
		Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถอัปเดตยอดหนี้ของลูกหนี้ได้"})
		return
	}

	// 4. Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถบันทึกข้อมูลได้"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "บันทึกรายการค้างชำระและอัปเดตยอดหนี้สำเร็จ",
		"sale_id": input.SaleID,
	})
}
