package routes

import (
	shopController "EazyStoreAPI/api/shops"

	"github.com/gin-gonic/gin"
)

func ShopRoutes(rg *gin.RouterGroup) {
	rg.POST("/shops", shopController.CreateShop)          // สร้างร้านค้า
	rg.GET("/shops", shopController.GetShopByUser)        // ดึงข้อมูลร้านค้า
	rg.PUT("/shops/:shop_id", shopController.UpdateShop)    // แก้ไขร้านค้า
	rg.DELETE("/shops/:shop_id", shopController.DeleteShop) // ลบร้านค้า
}