package routes

import (
	debtorController "EazyStoreAPI/controllers/debtor"
	paymentController "EazyStoreAPI/controllers/payment"

	"github.com/gin-gonic/gin"
)

func DebtorRoutes(rg *gin.RouterGroup) {
	rg.POST("/debtors", debtorController.CreateDebtor)                     // สร้างลูกหนี้
	rg.GET("/debtors", debtorController.GetDebtorByAll)                    // ดึงลูกหนี้ทั้งหมด
	rg.GET("/debtors/search", debtorController.GetDebtorBySearch)          // ค้นหาลูกหนี้
	rg.GET("/debtors/:id/history", debtorController.GetDebtorHistory)      // ประวัติการซื้อ/ค้างของลูกหนี้
	rg.GET("/debtors/:id/payments", paymentController.GetDebtorPaymentHistory) // ประวัติการชำระเงินของลูกหนี้
	rg.PUT("/debtors/:id", debtorController.UpdateDebtor)                  // แก้ไขข้อมูลลูกหนี้
}