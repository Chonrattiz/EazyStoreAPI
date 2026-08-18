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
		c.JSON(http.StatusBadRequest, gin.H{"error": "ข้อมูลร้านค้าไม่ครบถ้วนหรือรูปแบบไม่ถูกต้อง"})
		return
	}

	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ไม่พบข้อมูลยืนยันตัวตน (กรุณา Login)"})
		return
	}

	var userID int
	switch v := userIDValue.(type) {
	case float64:
		userID = int(v)
	case int:
		userID = v
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "รูปแบบ UserID ไม่ถูกต้อง"})
		return
	}

	shop := models.Shop{
		UserID:    userID,
		Name:      input.Name,
		Phone:     input.Phone,
		Address:   input.Address,
		ImgQrcode: input.ImgQrcode,
		ImgShop:   input.ImgShop,
		Pincode:   input.Pincode,
	}

	if err := database.DB.Create(&shop).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถบันทึกร้านค้าได้ กรุณาลองใหม่อีกครั้ง"})
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "กรุณาเข้าสู่ระบบก่อนใช้งาน"})
		return
	}

	var shops []models.Shop

	result := database.DB.Where("user_id = ?", userId).Find(&shops)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถดึงข้อมูลร้านค้าได้"})
		return
	}

	if len(shops) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "ไม่พบข้อมูลร้านค้า", "data": []string{}})
		return
	}

	c.JSON(http.StatusOK, shops)
}

// DeleteShop godoc
// @Summary      ลบร้านค้า
// @Description  ลบร้านค้าของ User id และ Shop id ที่ส่งเข้ามา
// @Tags         Shop
// @Accept       json
// @Produce      json
// @Param        shop_id   path      int  true  "Shop ID"
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]string "Shop not found"
// @Failure      500  {object}  map[string]string "Internal Error"
// @Router       /api/deleteShop/{shop_id} [delete]
func DeleteShop(c *gin.Context) {
	shopID := c.Param("shop_id")

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ไม่พบข้อมูลผู้ใช้ กรุณาเข้าสู่ระบบใหม่"})
		return
	}

	var shop models.Shop
	result := database.DB.Where("shop_id = ? AND user_id = ?", shopID, userID).Delete(&shop)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถลบร้านค้าได้ กรุณาลองใหม่อีกครั้ง"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "ไม่พบร้านค้า หรือ คุณไม่มีสิทธิ์ลบร้านนี้",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "ลบร้านค้าเรียบร้อยแล้ว",
	})
}

// UpdateShop godoc
// @Summary      แก้ไขข้อมูลร้านค้า
// @Description  แก้ไขเฉพาะข้อมูลที่ส่งมา (Field ไหนไม่ส่งมา จะใช้ค่าเดิม)
// @Tags         Shop
// @Accept       json
// @Produce      json
// @Param        shop_id  path      int              true  "Shop ID"
// @Param  		 shop  body  models.UpdateShopInput  true  "ข้อมูลที่ต้องการแก้"
// @Security     BearerAuth
// @Success      200      {object}  map[string]interface{}
// @Failure      400      {object}  map[string]string "Bad Request"
// @Router       /api/updateShop/{shop_id} [put]
func UpdateShop(c *gin.Context) {
	shopID := c.Param("shop_id")
	userID, _ := c.Get("user_id")

	var input models.UpdateShopInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "รูปแบบข้อมูลที่ส่งมาไม่ถูกต้อง"})
		return
	}

	var shop models.Shop
	if err := database.DB.Where("shop_id = ? AND user_id = ?", shopID, userID).First(&shop).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบร้านค้า หรือ ไม่มีสิทธิ์แก้ไข"})
		return
	}

	if input.Name != nil {
		shop.Name = *input.Name
	}
	if input.Phone != nil {
		shop.Phone = *input.Phone
	}
	if input.Address != nil {
		shop.Address = *input.Address
	}
	if input.ImgQRCode != nil {
		shop.ImgQrcode = *input.ImgQRCode
	}
	if input.ImgShop != nil {
		shop.ImgShop = *input.ImgShop
	}
	if input.PinCode != nil {
		shop.Pincode = *input.PinCode
	}


	if err := database.DB.Save(&shop).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "บันทึกไม่สำเร็จ"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "อัปเดตข้อมูลเรียบร้อย",
		"data":    shop,
	})
}
