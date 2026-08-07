package controllers

import (
	"EazyStoreAPI/database"
	"EazyStoreAPI/middleware"
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
	if !middleware.RequireShopAccess(c, input.ShopID) {
		return
	}

	var duplicateCount int64
	database.DB.Model(&models.Category{}).
		Where("shop_id = ? AND name = ?", input.ShopID, input.Name).
		Count(&duplicateCount)
	if duplicateCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "มีหมวดหมู่ชื่อ \"" + input.Name + "\" อยู่แล้ว"})
		return
	}

	category := models.Category{ShopID: input.ShopID, Name: input.Name, Status: true}
	if err := database.DB.Create(&category).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "ไม่สามารถเพิ่มหมวดหมู่ได้"})
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
	if !middleware.RequireShopAccess(c, input.ShopID) {
		return
	}

	var duplicateCount int64
	database.DB.Model(&models.Category{}).
		Where("shop_id = ? AND name = ? AND category_id <> ?", input.ShopID, input.Name, categoryID).
		Count(&duplicateCount)
	if duplicateCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "มีหมวดหมู่ชื่อ \"" + input.Name + "\" อยู่แล้ว"})
		return
	}

	result := database.DB.Model(&models.Category{}).
		Where("category_id = ? AND shop_id = ? AND status = ?", categoryID, input.ShopID, true).
		Update("name", input.Name)
	if result.Error != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "ไม่สามารถแก้ไขหมวดหมู่ได้"})
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
	if !middleware.RequireShopAccess(c, shopID) {
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

func GetInactiveCategories(c *gin.Context) {
	shopID, ok := middleware.RequireShopIDQuery(c)
	if !ok {
		return
	}

	var categories []models.Category
	if err := database.DB.
		Where("shop_id = ? AND status = ?", shopID, false).
		Order("category_id DESC").
		Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถดึงหมวดหมู่ที่ปิดใช้งานได้"})
		return
	}
	c.JSON(http.StatusOK, categories)
}

func RestoreCategory(c *gin.Context) {
	categoryID, err := strconv.Atoi(c.Param("id"))
	if err != nil || categoryID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category_id ไม่ถูกต้อง"})
		return
	}

	var input models.RestoreCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาระบุ shop_id"})
		return
	}
	if !middleware.RequireShopAccess(c, input.ShopID) {
		return
	}

	var category models.Category
	if err := database.DB.
		Where("category_id = ? AND shop_id = ? AND status = ?", categoryID, input.ShopID, false).
		First(&category).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบหมวดหมู่ที่ปิดใช้งานของร้านนี้"})
		return
	}

	var activeDuplicate int64
	database.DB.Model(&models.Category{}).
		Where("shop_id = ? AND status = ? AND name = ? AND category_id <> ?",
			input.ShopID, true, category.Name, categoryID).
		Count(&activeDuplicate)
	if activeDuplicate > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error": "มีหมวดหมู่ชื่อ \"" + category.Name + "\" ใช้งานอยู่แล้ว กรุณาเปลี่ยนชื่อหมวดที่ใช้งานอยู่ก่อนกู้คืน",
		})
		return
	}

	result := database.DB.Model(&models.Category{}).
		Where("category_id = ? AND shop_id = ? AND status = ?", categoryID, input.ShopID, false).
		Update("status", true)
	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "Duplicate") ||
			strings.Contains(result.Error.Error(), "duplicate") {
			c.JSON(http.StatusConflict, gin.H{
				"error": "มีหมวดหมู่ชื่อ \"" + category.Name + "\" ใช้งานอยู่แล้ว กรุณาเปลี่ยนชื่อหมวดที่ใช้งานอยู่ก่อนกู้คืน",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถกู้คืนหมวดหมู่ได้"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบหมวดหมู่ที่ปิดใช้งานของร้านนี้"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "กู้คืนหมวดหมู่สำเร็จ", "status": "active"})
}

func MoveCategoryProducts(c *gin.Context) {
	sourceCategoryID, err := strconv.Atoi(c.Param("id"))
	if err != nil || sourceCategoryID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category_id ไม่ถูกต้อง"})
		return
	}

	var input models.MoveCategoryProductsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาระบุ shop_id และ target_category_id"})
		return
	}
	if input.TargetCategoryID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_category_id ไม่ถูกต้อง"})
		return
	}
	if input.TargetCategoryID == sourceCategoryID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ไม่สามารถย้ายสินค้าไปหมวดหมู่เดิมได้"})
		return
	}
	if !middleware.RequireShopAccess(c, input.ShopID) {
		return
	}

	var sourceCategory models.Category
	if err := database.DB.
		Where("category_id = ? AND shop_id = ?", sourceCategoryID, input.ShopID).
		First(&sourceCategory).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบหมวดหมู่ต้นทางของร้านนี้"})
		return
	}

	var targetCategory models.Category
	if err := database.DB.
		Where("category_id = ? AND shop_id = ? AND status = ?",
			input.TargetCategoryID, input.ShopID, true).
		First(&targetCategory).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบหมวดหมู่ปลายทางของร้านนี้"})
		return
	}

	moveQuery := database.DB.Model(&models.Product{}).
		Where("shop_id = ? AND category_id = ?", input.ShopID, sourceCategoryID)
	if len(input.ProductIDs) > 0 {
		moveQuery = moveQuery.Where("product_id IN ?", input.ProductIDs)
	}

	var movedCount int64
	if err := moveQuery.Count(&movedCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถตรวจสอบสินค้าได้"})
		return
	}
	if movedCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ไม่พบสินค้าในหมวดหมู่นี้ที่จะย้าย"})
		return
	}

	updateQuery := database.DB.Model(&models.Product{}).
		Where("shop_id = ? AND category_id = ?", input.ShopID, sourceCategoryID)
	if len(input.ProductIDs) > 0 {
		updateQuery = updateQuery.Where("product_id IN ?", input.ProductIDs)
	}
	result := updateQuery.Update("category_id", input.TargetCategoryID)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถย้ายสินค้าได้"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":            "ย้ายสินค้าสำเร็จ",
		"moved_count":        result.RowsAffected,
		"from_category_id":   sourceCategoryID,
		"target_category_id": input.TargetCategoryID,
	})
}
