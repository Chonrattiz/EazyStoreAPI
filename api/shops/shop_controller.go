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
// @Security     BearerAuth
// @Param        Shop body models.Shop true "ข้อมูลร้านค้า"
// @Success      200  {object} models.Shop
// @Failure      400  {object} map[string]string
// @Router       /api/createShop [post]
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

// GetShopByUser godoc
// @Summary      ดูร้านค้า
// @Description  ดูร้านค้าของ User ที่ Login อยู่ (ดึง ID จาก Token)
// @Tags         Shop
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   models.Shop
// @Failure      404  {object}  map[string]string "Shop not found"
// @Failure      500  {object}  map[string]string "Internal Error"
// @Router       /api/getShop [get]
func GetShopByUser(c *gin.Context) {
	userId, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var shops []models.Shop

	result := database.DB.Where("user_id = ?", userId).Find(&shops)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	// เช็คว่ามีร้านไหม (Optional: ถ้าอยากให้ return 404 เมื่อไม่มีร้านเลย)
	if len(shops) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "No shops found", "data": []string{}})
		return
	}

	c.JSON(http.StatusOK, shops)
}
