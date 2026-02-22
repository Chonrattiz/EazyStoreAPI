package routes

import (
	saleController "EazyStoreAPI/api/sales"

	"github.com/gin-gonic/gin"
)

func SaleRoutes(rg *gin.RouterGroup) {
	rg.POST("/sales", saleController.CreateSale)               // บันทึกการขายปกติ
	rg.POST("/sales/credit", saleController.CreateCreditSale)  // บันทึกการขายเชื่อ (ค้างชำระ)
}