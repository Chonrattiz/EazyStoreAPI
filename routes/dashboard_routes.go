package routes

import (
	dashboadController "EazyStoreAPI/controllers/dashboad"

	"github.com/gin-gonic/gin"
)

func DashboardRoutes(rg *gin.RouterGroup) {
	// ดึงสรุปยอดขายสำหรับ Dashboard
	rg.GET("/dashboard/sales-summary", dashboadController.GetSalesSummary) 
	rg.GET("/dashboard/transactions", dashboadController.GetTransactionsDetail)
	rg.GET("/dashboard/product-details", dashboadController.GetProductSalesDetail)
	rg.GET("/dashboard/advanced-report", dashboadController.GetAdvancedReport)
	rg.GET("/dashboard/sale-items", dashboadController.GetSaleItems)
}