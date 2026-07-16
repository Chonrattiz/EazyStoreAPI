package controllers

import (
	"EazyStoreAPI/database"
	"EazyStoreAPI/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func CreateCategory(c *gin.Context) {
	var input models.CreateCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาระบุ shop_id และชื่อหมวดหมู่"})
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาระบุชื่อหมวดหมู่"})
		return
	}

	category := models.Category{ShopID: input.ShopID, Name: input.Name, Status: true}
	if err := database.DB.Create(&category).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "ไม่สามารถเพิ่มหมวดหมู่ได้: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, category)
}

func UpdateCategory(c *gin.Context) {
	categoryID, err := strconv.Atoi(c.Param("id"))
	if err != nil || categoryID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category_id ไม่ถูกต้อง"})
		return
	}
	var input models.UpdateCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาระบุ shop_id และชื่อหมวดหมู่"})
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาระบุชื่อหมวดหมู่"})
		return
	}

	result := database.DB.Model(&models.Category{}).
		Where("category_id = ? AND shop_id = ? AND status = ?", categoryID, input.ShopID, true).
		Update("name", input.Name)
	if result.Error != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "ไม่สามารถแก้ไขหมวดหมู่ได้: " + result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบหมวดหมู่ของร้านนี้"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "แก้ไขหมวดหมู่สำเร็จ"})
}

func DeleteCategory(c *gin.Context) {
	categoryID, err := strconv.Atoi(c.Param("id"))
	shopID, shopErr := strconv.Atoi(c.Query("shop_id"))
	if err != nil || categoryID <= 0 || shopErr != nil || shopID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาระบุ category_id และ shop_id ที่ถูกต้อง"})
		return
	}

	result := database.DB.Model(&models.Category{}).
		Where("category_id = ? AND shop_id = ? AND status = ?", categoryID, shopID, true).
		Update("status", false)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถปิดใช้งานหมวดหมู่ได้"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบหมวดหมู่ของร้านนี้"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ปิดใช้งานหมวดหมู่สำเร็จ", "status": "inactive"})
}
