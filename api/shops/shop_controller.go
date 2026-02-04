package controllers

import (
	"EazyStoreAPI/database"
	"net/http"

	"EazyStoreAPI/models"

	"github.com/gin-gonic/gin"
)

// CreateShop godoc
// @Summary      เพิ่มร้านค้า
// @Description  สร้างร้านค้าใหม่ลงในฐานข้อมูล
// @Tags         Shop
// @Accept       json
// @Produce      json
// @Param        Shop body models.Shop true "ข้อมูลร้านค้า"
// @Success      200  {object} models.Shop
// @Failure      400  {object} map[string]string
// @Router       /CreateShop [post]
func CreateShop(c *gin.Context) {
	var input models.Shop

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	shop := models.Shop{
		UserID:    input.UserID,
		Name:      input.Name,
		Phone:     input.Phone,
		Address:   input.Address,
		ImgQrcode: input.ImgQrcode,
		ImgShop:   input.ImgShop,
		Pincode:   input.Pincode,
	}

	if err := database.DB.Create(&shop).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถบันทึกร้านค้าได้: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "บันทึกร้านค้าสำเร็จ",
		"data":    shop,
	})
}
