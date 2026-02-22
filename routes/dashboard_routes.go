package routes

import (
	dashboadController "EazyStoreAPI/api/dashboad"

	"github.com/gin-gonic/gin"
)

func DashboardRoutes(rg *gin.RouterGroup) {
	// ดึงสรุปยอดขายสำหรับ Dashboard
	rg.GET("/dashboard/sales-summary", dashboadController.GetSalesSummary) 
}