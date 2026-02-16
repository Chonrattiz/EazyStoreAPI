package controller

import (
	"EazyStoreAPI/database"
	"EazyStoreAPI/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// PaymentDebt godoc
// @Summary      ชำระหนี้ (พร้อมตรวจสอบ PIN ร้านค้า)
// @Description  API นี้ใช้สำหรับบันทึกการชำระหนี้ โดยจะทำการตรวจสอบ PIN Code 6 หลักของร้านค้าก่อนตัดยอดหนี้และบันทึกประวัติ
// @Tags         Debtor
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body models.PayDebtRequest true "ข้อมูลการชำระหนี้และ PIN Code"
// @Success      200  {object}  map[string]interface{} "บันทึกสำเร็จ (return JSON object)"
// @Failure      400  {object}  map[string]string      "ข้อมูลไม่ครบถ้วน (Bad Request)"
// @Failure      401  {object}  map[string]string      "รหัส PIN ไม่ถูกต้อง (Unauthorized)"
// @Failure      404  {object}  map[string]string      "ไม่พบร้านค้า หรือ ลูกหนี้ (Not Found)"
// @Failure      500  {object}  map[string]string      "ข้อผิดพลาดภายในเซิร์ฟเวอร์ (Internal Server Error)"
// @Router       /api/paymentDebt [post]
func PaymentDebt(c *gin.Context) {
	var input models.PayDebtRequest

	// 1. รับค่า Json จาก Flutter
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ข้อมูลไม่ครบถ้วน หรือรูปแบบไม่ถูกต้อง"})
		return
	}

	// 2. 🔥 ตรวจสอบ PIN Code ของร้านค้า
	var shop models.Shop
	// ค้นหาร้านค้าด้วย ShopID และ PinCode
	if err := database.DB.Where("shop_id = ? AND pin_code = ?", input.ShopID, input.PinCode).First(&shop).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "รหัส PIN ไม่ถูกต้อง"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "เกิดข้อผิดพลาดในการตรวจสอบร้านค้า"})
		}
		return
	}

	// --- เริ่ม Transaction (เพื่อให้ข้อมูล Debt และ Payment ตรงกันเสมอ) ---
	tx := database.DB.Begin()

	// 3. ดึงข้อมูลลูกหนี้ปัจจุบัน
	var debtor models.Debtor
	if err := tx.Where("debtor_id = ?", input.DebtorID).First(&debtor).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบข้อมูลลูกหนี้"})
		return
	}

	// 4. คำนวณยอดหนี้คงเหลือใหม่
	newTotalDebt := debtor.CurrentDebt - input.AmountPaid

	// 5. บันทึกประวัติการจ่ายเงิน (DebtPayment)
	newPayment := models.DebtPayment{
		DebtorID:      input.DebtorID,
		AmountPaid:    input.AmountPaid,
		PaymentMethod: input.PaymentMethod,
		CurrentDebt:   newTotalDebt, // บันทึกยอดคงเหลือ ณ ขณะนั้น
		PaymentDate:   time.Now(),
		RecordedBy:    input.PayWith,
	}

	if err := tx.Create(&newPayment).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "บันทึกประวัติการจ่ายไม่สำเร็จ"})
		return
	}

	// 6. อัปเดตยอดหนี้ในตารางลูกหนี้ (Debtors)
	if err := tx.Model(&debtor).Update("current_debt", newTotalDebt).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "อัปเดตยอดหนี้ไม่สำเร็จ"})
		return
	}

	// Commit Transaction (ยืนยันการบันทึกทั้งหมด)
	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message":    "บันทึกการชำระเงินเรียบร้อย",
		"new_debt":   newTotalDebt,
		"payment_id": newPayment.PaymentID,
	})
}
