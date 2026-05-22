package routes

import (
	dashboadController "EazyStoreAPI/api/dashboad"

	"github.com/gin-gonic/gin"
)

func DashboardRoutes(rg *gin.RouterGroup) {
	// ดึงสรุปยอดขายสำหรับ Dashboard
	rg.GET("/dashboard/sales-summary", dashboadController.GetSalesSummary) 
	rg.GET("/dashboard/transactions", dashboadController.GetTransactionsDetail)
	rg.GET("/dashboard/product-details", dashboadController.GetProductSalesDetail)
	rg.GET("/dashboard/advanced-report", dashboadController.GetAdvancedReport)
}