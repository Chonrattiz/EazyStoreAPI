package routes

import (
	paymentController "EazyStoreAPI/api/payment"

	"github.com/gin-gonic/gin"
)

func PaymentRoutes(rg *gin.RouterGroup) {
	rg.POST("/payments", paymentController.PaymentDebt) // ชำระหนี้ (ใช้เป็น /payments หรือ /payments/debt ก็ได้)
}