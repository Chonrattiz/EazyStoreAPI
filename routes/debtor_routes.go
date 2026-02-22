package routes

import (
	debtorController "EazyStoreAPI/api/debtor"

	"github.com/gin-gonic/gin"
)

func DebtorRoutes(rg *gin.RouterGroup) {
	rg.POST("/debtors", debtorController.CreateDebtor)            // สร้างลูกหนี้
	rg.GET("/debtors", debtorController.GetDebtorByAll)           // ดึงลูกหนี้ทั้งหมด
	rg.GET("/debtors/search", debtorController.GetDebtorBySearch) // ค้นหาลูกหนี้
	rg.GET("/debtors/:id/history", debtorController.GetDebtorHistory) // ประวัติลูกหนี้
}