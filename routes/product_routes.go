package routes

import (
	productController "EazyStoreAPI/api/products"

	"github.com/gin-gonic/gin"
)

func ProductRoutes(rg *gin.RouterGroup) {
	rg.POST("/products", productController.CreateProduct)           // สร้างสินค้า
	rg.GET("/products", productController.GetProductsByShop)        // ดึงสินค้าทั้งหมด
	rg.GET("/products/search", productController.GetProductBySearch) // ค้นหาสินค้า
	rg.PUT("/products/:id", productController.UpdateProduct)        // แก้ไขสินค้า
	rg.DELETE("/products/:id", productController.DeleteProduct)     // ลบสินค้า
	
	// Action พิเศษ
	rg.PUT("/products/stock", productController.UpdateStock)        // อัปเดตสต็อก
	rg.GET("/products/null-barcode", productController.GetNullBarcode) // หาสินค้าไม่มีบาร์โค้ด
	
	// หมวดหมู่ 
	rg.GET("/categories", productController.GetCategories)
}