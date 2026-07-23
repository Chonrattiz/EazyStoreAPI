package routes

import (
	productController "EazyStoreAPI/controllers/products"

	"github.com/gin-gonic/gin"
)

func ProductRoutes(rg *gin.RouterGroup) {
	rg.POST("/products", productController.CreateProduct)            // สร้างสินค้า
	rg.GET("/products", productController.GetProductsByShop)         // ดึงสินค้าทั้งหมด
	rg.GET("/products/search", productController.GetProductBySearch) // ค้นหาสินค้า
	rg.PUT("/products/:id", productController.UpdateProduct)         // แก้ไขสินค้า
	rg.DELETE("/products/:id", productController.DeleteProduct)      // ลบสินค้า

	rg.PUT("/products/stock", productController.UpdateStock)           // อัปเดตสต็อก
	rg.GET("/products/null-barcode", productController.GetNullBarcode) // หาสินค้าไม่มีบาร์โค้ด

	// หน่วยขายเพิ่มเติม (ลัง/แพ็ค)
	rg.POST("/products/:id/units", productController.CreateProductUnit)
	rg.PUT("/products/units/:unitId", productController.UpdateProductUnit)
	rg.DELETE("/products/units/:unitId", productController.DeleteProductUnit)

	// หมวดหมู่
	rg.GET("/categories/inactive", productController.GetInactiveCategories)
	rg.GET("/categories", productController.GetCategories)
	rg.POST("/categories", productController.CreateCategory)
	rg.PUT("/categories/:id/restore", productController.RestoreCategory)
	rg.PUT("/categories/:id/move-products", productController.MoveCategoryProducts)
	rg.PUT("/categories/:id", productController.UpdateCategory)
	rg.DELETE("/categories/:id", productController.DeleteCategory)
}
