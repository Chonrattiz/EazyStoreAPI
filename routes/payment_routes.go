package routes

import (
	paymentController "EazyStoreAPI/controllers/payment"

	"github.com/gin-gonic/gin"
)

func PaymentRoutes(rg *gin.RouterGroup) {
	rg.POST("/payments", paymentController.PaymentDebt) // บันทึกการชำระหนี้
	// ประวัติการชำระของลูกหนี้ย้ายไปเป็น sub-resource ของ debtor แล้ว
	// ดูที่ GET /debtors/:id/payments (ใน debtor_routes.go)
}
