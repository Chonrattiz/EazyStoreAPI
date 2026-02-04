package controllers

import (
	"EazyStoreAPI/database"
	"net/http"
	

	"EazyStoreAPI/models"

	"github.com/gin-gonic/gin"
)

// CreateProduct godoc
// @Summary      เพิ่มสินค้า
// @Description  สร้างรายการสินค้าใหม่ลงในฐานข้อมูล
// @Tags         Product
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        product body models.Product true "ข้อมูลสินค้า"
// @Success      200  {object} models.Product
// @Failure      400  {object} map[string]string
// @Router       /api/createProduct [post]
func CreateProduct(c *gin.Context) {
	var input models.Product

	// 2. Validate Input
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 3. Map Input to Model (เรียกใช้ models.Product)
	product := models.Product{
		ShopID:     input.ShopID,
		CategoryID: input.CategoryID,
		Name:       input.Name,
		Barcode:    input.Barcode,
		ImgProduct: input.ImgProduct,
		SellPrice:  input.SellPrice,
		CostPrice:  input.CostPrice,
		Stock:      input.Stock,
		Unit:       input.Unit,
		Status:     input.Status,
		// ไม่ต้องใส่ ProductCode/ProductSeq เพราะ Trigger ใน DB จัดการให้
	}

	// 4. Insert into Database (เรียกใช้ database.DB หรือชื่อตัวแปรที่คุณตั้งใน package database)
	if err := database.DB.Create(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถบันทึกสินค้าได้: " + err.Error()})
		return
	}

	// 5. Return Success
	c.JSON(http.StatusOK, gin.H{
		"message": "บันทึกสินค้าสำเร็จ",
		"data":    product,
	})
}
