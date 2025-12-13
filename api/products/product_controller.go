package controllers

import (
	"EazyStoreAPI/database"
	models "EazyStoreAPI/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetProducts godoc
// @Summary      ดึงรายการสินค้า
// @Description  แสดงสินค้าทั้งหมดในระบบ
// @Tags         Products
// @Accept       json
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /products [get]
func GetProducts(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Hello Swagger!",
		"data":    []string{"Apple", "Banana"},
	})
}

// UpdateProduct godoc
// @Summary      อัปเดตข้อมูลสินค้าแล้วครับ
// @Description  แก้ไขข้อมูลสินค้าตาม ID
// @Tags         Products
// @Accept       json
// @Produce      json
// @Param        id      path      int              true  "Product ID"
// @Param        product body      models.Product   true  "ข้อมูลสินค้า"
// @Success      200     {object}  models.Product
// @Failure      400     {object}  map[string]string
// @Failure      404     {object}  map[string]string
// @Router       /products/{id} [put]
func UpdateProduct(c *gin.Context) {
	id := c.Param("id")

	var product models.Product
	if err := database.DB.First(&product, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Product not found",
		})
		return
	}

	var input models.Product
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	product.Name = input.Name
	product.Price = input.Price
	product.Description = input.Description
	product.Stock = input.Stock

	database.DB.Save(&product)

	c.JSON(http.StatusOK, product)
}
