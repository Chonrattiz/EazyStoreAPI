package routes

import (
	OrderListController "EazyStoreAPI/controllers/OrderList"

	"github.com/gin-gonic/gin"
)

func OrderListRoutes(rg *gin.RouterGroup) {
	rg.POST("/orderlist", OrderListController.ExportOrderPDF)

}
