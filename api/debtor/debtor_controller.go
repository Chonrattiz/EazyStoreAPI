package controllers

import (
	"EazyStoreAPI/database"
	"net/http"

	"EazyStoreAPI/models"

	"strings"

	"github.com/gin-gonic/gin"
)

// CreateDebtor godoc
// @Summary      เพิ่มลูกหนี้
// @Description  สร้างลูกหนี้ใหม่ลงในฐานข้อมูล
// @Tags         Debtor
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Debtor body models.Debtor true "ข้อมูลลูกหนี้"
// @Success      200  {object} models.Debtor
// @Failure      400  {object} map[string]string
// @Router       /api/createDebtor [post]
func CreateDebtor(c *gin.Context) {
	var input models.Debtor

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ข้อมูลไม่ถูกต้อง: " + err.Error()})
		return
	}

	if err := database.DB.Create(&input).Error; err != nil {
		// เช็คว่า Error คือเบอร์ซ้ำหรือไม่ (MySQL Error 1062)
		if strings.Contains(err.Error(), "1062") || strings.Contains(err.Error(), "Duplicate entry") {
			c.JSON(http.StatusConflict, gin.H{"error": "เบอร์โทรศัพท์นี้มีในระบบของร้านท่านแล้ว"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถบันทึกได้"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "บันทึกลูกหนี้สำเร็จ",
		"data":    input,
	})
}

// GetDebtorBySearch godoc
// @Summary      ค้นหาลูกหนี้
// @Description  ค้นหาลูกหนี้ด้วย Keyword (รองรับทั้ง ชื่อลูกหนี้, เบอร์โทร)
// @Tags         Debtor
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        keyword    query    string   true   "คำค้นหา (ระบุ  ชื่อ หรือ เบอร์โทร)"
// @Param        shop_id   query    int     true  "รหัสร้านค้า (Shop ID)"
// @Success      200  {array}  models.Debtor
// @Failure      400  {object}  map[string]string "ระบุพารามิเตอร์ไม่ครบ"
// @Failure      404  {object}  map[string]string "ไม่พบข้อมูลลูกหนี้"
// @Router       /api/debtor/search [get]
func GetDebtorBySearch(c *gin.Context) {
	keyword := c.Query("keyword")
	shopID := c.Query("shop_id")

	if keyword == "" || shopID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาระบุ keyword และ shop_id"})
		return
	}

	var debtors []models.Debtor

	result := database.DB.Where("shop_id = ?", shopID).
		Where(database.DB.Where("phone LIKE ?", "%"+keyword+"%").Or("name LIKE ?", "%"+keyword+"%")).
		Find(&debtors)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "เกิดข้อผิดพลาดในการดึงข้อมูล"})
		return
	}
	if len(debtors) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบข้อมูลลูกหนี้"})
		return
	}

	c.JSON(http.StatusOK, debtors)
}
