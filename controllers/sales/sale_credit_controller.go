package controllers

import (
	"EazyStoreAPI/database"
	"EazyStoreAPI/middleware"
	"EazyStoreAPI/models"
	"net/http"
	"time"

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

	// จ่ายครบ/จ่ายเกิน = ไม่มียอดค้าง ถ้าปล่อยผ่านส่วนเกินจะถูกเอาไปหักหนี้ก้อนเก่าโดยไม่ตั้งใจ
	if input.Pay >= input.NetPrice {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "รายการนี้จ่ายครบแล้ว ไม่มียอดค้างชำระ กรุณาบันทึกเป็นการขายเงินสดแทน",
		})
		return
	}

	// 2.5 shop_id มาจาก body ต้องตรวจว่าเป็นร้านของ user ที่ล็อกอินอยู่จริง
	// ด้านล่างใช้ input.ShopID กรองลูกหนี้/สินค้าอยู่แล้ว แต่ถ้าไม่ตรวจตรงนี้
	// ก็แค่สวมเป็นร้านอื่นทั้งบิลแทน (สร้างหนี้ให้ลูกหนี้ร้านอื่น + ตัดสต็อกร้านอื่น)
	if !middleware.RequireShopAccess(c, input.ShopID) {
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

		// ค. ตรวจสอบสินค้า เช็คสต๊อก และทำการตัดสต๊อก
		for _, item := range input.SaleItems {
			var product models.Product
			// เช็คว่าสินค้ามีอยู่จริงและเป็นของร้านนี้
			if err := tx.Where("product_id = ? AND shop_id = ?", item.ProductID, input.ShopID).First(&product).Error; err != nil {
				return fmt.Errorf("ไม่พบสินค้ารหัส %d หรือไม่ใช่สินค้าของร้านคุณ", item.ProductID)
			}

			// เช็คว่าสต๊อกพอขายหรือไม่
			if product.Stock < item.Amount {
				return fmt.Errorf("สินค้า '%s' มีสต๊อกไม่พอ (คงเหลือ %d ชิ้น, ต้องการ %d ชิ้น)", product.Name, product.Stock, item.Amount)
			}

			// ทำการตัดสต๊อก
			if err := tx.Model(&models.Product{}).
				Where("product_id = ? AND shop_id = ?", item.ProductID, input.ShopID).
				UpdateColumn("stock", gorm.Expr("stock - ?", item.Amount)).Error; err != nil {
				return fmt.Errorf("ไม่สามารถตัดสต๊อกสินค้า '%s' ได้", product.Name)
			}
		}

		// ง. บันทึกข้อมูลการขาย (Sales & SaleItems)
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
			return errors.New("ไม่สามารถบันทึกรายการขายได้")
		}

		// จ. อัปเดตยอดหนี้ปัจจุบัน (current_debt)
		if err := tx.Table("debtors").
			Where("debtor_id = ? AND shop_id = ?", input.DebtorID, input.ShopID).
			UpdateColumn("current_debt", gorm.Expr("current_debt + ?", amountToCharge)).
			Error; err != nil {
			return errors.New("ไม่สามารถอัปเดตยอดหนี้ของลูกหนี้ได้")
		}

		return nil
	})

	// 4. จัดการผลลัพธ์ (ข้อความ error ใน Transaction เป็นภาษาไทยทั้งหมดแล้ว ส่งกลับได้เลย)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "บันทึกรายการค้างชำระและอัปเดตยอดหนี้สำเร็จ",
		"sale_id": input.SaleID,
	})
}
