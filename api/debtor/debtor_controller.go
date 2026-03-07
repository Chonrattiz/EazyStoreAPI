package controllers

import (
	"EazyStoreAPI/database"
	"net/http"

	"EazyStoreAPI/models"

	"strings"
	"math"

	"fmt"
	"strconv"

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

// GetDebtorByAll godoc
// @Summary      ดึงข้อมูลลูกหนี้
// @Description  ดึงข้อมูลลูกหนี้ทั้งหมด ของ shopid นั้น
// @Tags         Debtor
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        shop_id   query    int     true  "รหัสร้านค้า (Shop ID)"
// @Success      200  {array}  models.Debtor
// @Failure      400  {object}  map[string]string "ระบุพารามิเตอร์ไม่ครบ"
// @Failure      404  {object}  map[string]string "ไม่พบข้อมูลลูกหนี้"
// @Router       /api/debtor [get]
func GetDebtorByAll(c *gin.Context) {
	shopID := c.Query("shop_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search") // เพิ่มค้นหาด้วย

	if shopID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาระบุ shop_id"})
		return
	}

	var debtors []models.Debtor
	var totalItems int64

	// สร้าง Query พื้นฐาน
	query := database.DB.Model(&models.Debtor{}).Where("shop_id = ?", shopID)

	// Filter ค้นหาชื่อหรือเบอร์โทร
	if search != "" {
		query = query.Where("name LIKE ? OR phone LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// นับจำนวนรวม
	query.Count(&totalItems)

	// Pagination และเรียงตาม ID (ใหม่ไปเก่า)
	offset := (page - 1) * limit
	result := query.Order("debtor_id DESC").Limit(limit).Offset(offset).Find(&debtors)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "เกิดข้อผิดพลาด"})
		return
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))

	// ส่งกลับรูปแบบเดียวกับ Product
	c.JSON(http.StatusOK, gin.H{
		"items":        debtors,
		"total_items":  totalItems,
		"total_pages":  totalPages,
		"current_page": page,
	})
}

// GetDebtorHistory godoc
// @Summary      ดึงประวัติการติดหนี้ของลูกหนี้
// @Description  ดึงข้อมูลลูกหนี้ ยอดคงเหลือ และประวัติบิลที่ยังค้างชำระพร้อมรายการสินค้า
// @Tags         Debtor
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "รหัสลูกหนี้ (Debtor ID)"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string "ข้อมูลไม่ถูกต้อง"
// @Failure      404  {object}  map[string]string "ไม่พบข้อมูลลูกหนี้"
// @Failure      500  {object}  map[string]string "เกิดข้อผิดพลาดในเซิร์ฟเวอร์"
// @Router       /api/debtor/{id}/history [get]
func GetDebtorHistory(c *gin.Context) {
	// รับค่า ID ลูกหนี้จาก URL Path
	idParam := c.Param("id")
	debtorID, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "รหัสลูกหนี้ไม่ถูกต้อง"})
		return
	}

	// 1. ค้นหาข้อมูลลูกหนี้
	var debtor models.Debtor
	if err := database.DB.First(&debtor, debtorID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบข้อมูลลูกหนี้"})
		return
	}

	// 2. ค้นหาบิลทั้งหมดของลูกหนี้ที่ยังจ่ายไม่ครบ (ติดหนี้)
	var sales []models.Sale
	if err := database.DB.Preload("SaleItems").
		Where("debtor_id = ? AND net_price > pay", debtorID).
		Order("created_at desc, created_time desc").
		Find(&sales).Error; err != nil {
		fmt.Println("🚨 GORM ERROR:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "เกิดข้อผิดพลาดในการดึงประวัติบิล"})
		return
	}

	// 3. รวบรวม ProductID เพื่อไปดึงชื่อและหน่วยสินค้ามาแสดง
	var productIDs []int
	for _, sale := range sales {
		for _, item := range sale.SaleItems {
			productIDs = append(productIDs, item.ProductID)
		}
	}

	// สร้าง Map เก็บข้อมูลสินค้าทั้งตัว (เพื่อให้ดึงได้ทั้ง ชื่อ และ หน่วย)
	productMap := make(map[int]models.Product)
	if len(productIDs) > 0 {
		var products []models.Product
		// สมมติว่าในฐานข้อมูลมีคอลัมน์ unit หรือใน Struct Product มีฟิลด์ Unit
		database.DB.Select("product_id, name, unit").Where("product_id IN ?", productIDs).Find(&products)
		for _, p := range products {
			productMap[p.ProductID] = p
		}
	}

	// 4. ประกอบร่างข้อมูลเป็น JSON (โดยใช้ Array ของ gin.H)
	var histories []gin.H // เตรียม Array ไว้เก็บข้อมูลบิล
	thaiMonths := []string{"", "ม.ค.", "ก.พ.", "มี.ค.", "เม.ย.", "พ.ค.", "มิ.ย.", "ก.ค.", "ส.ค.", "ก.ย.", "ต.ค.", "พ.ย.", "ธ.ค."}

	for _, sale := range sales {

		// 1. จัดการวันที่ (แปลงเป็น พ.ศ. และเดือนไทย)
		day := sale.CreatedAt.Day()
		month := int(sale.CreatedAt.Month())
		year := sale.CreatedAt.Year() + 543
		// จัดฟอร์แมตวันที่ เช่น "16 ก.พ. 2569" (ถ้าเลขวันหลักเดียวจะมี 0 นำหน้า เช่น 05)
		dateStr := fmt.Sprintf("%02d %s %d", day, thaiMonths[month], year)

		// 2. จัดการเวลา (ดึงค่าออกจาก Pointer เพื่อแก้ปัญหา 0xc00)
		timeStr := ""
		if sale.CreatedTime != nil {
			t := *sale.CreatedTime // ใส่เครื่องหมาย * เพื่อดึงค่าข้อความออกมาจาก Pointer
			if len(t) >= 5 {
				timeStr = t[:5] // เอาแค่ HH:mm
			} else {
				timeStr = t
			}
		}

		// นำมารวมกัน จะได้เช่น "16 ก.พ. 2569 14:04"
		dateTimeStr := fmt.Sprintf("%s %s", dateStr, timeStr)

		var items []gin.H
		for _, item := range sale.SaleItems {
			name := "สินค้าไม่ทราบชื่อ"
			unit := "ชิ้น" // ค่าเริ่มต้นเผื่อไม่มีระบุ

			if p, ok := productMap[item.ProductID]; ok {
				name = p.Name
				if p.Unit != "" {
					unit = p.Unit
				}
			}

			items = append(items, gin.H{
				"name":  name,
				"qty":   item.Amount,
				"price": item.TotalPrice,
				"unit":  unit,
			})
		}

		// ใส่ข้อมูลบิลลง Array
		histories = append(histories, gin.H{
			"order_id":         sale.SaleID,
			"date":             dateTimeStr,
			"total_amount":     sale.NetPrice,
			"paid_amount":      sale.Pay,
			"remaining_amount": sale.NetPrice - sale.Pay,
			"items":            items,
		})
	}

	// กันเหนียว กรณีไม่มีประวัติให้เป็น [] ว่างๆ จะได้ไม่เป็น null ตอนส่ง JSON
	if histories == nil {
		histories = []gin.H{}
	}

	// 5. ส่งกลับ JSON ทันทีด้วยโครงสร้างที่ต้องการ
	c.JSON(http.StatusOK, gin.H{
		"debtor_id":     debtor.DebtorID,
		"name":          debtor.Name,
		"phone":         debtor.Phone,
		"address":       debtor.Address,
		"current_debt":  debtor.CurrentDebt,
		"credit_limit":  debtor.CreditLimit,
		"credit_remain": debtor.CreditLimit - debtor.CurrentDebt,
		"histories":     histories,
	})
}

// UpdateDebtor godoc
// @Summary      แก้ไขข้อมูลลูกหนี้
// @Description  แก้ไขข้อมูลลูกหนี้ตาม ID (อัปเดตเฉพาะฟิลด์ที่ส่งมา)
// @Tags         Debtor
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "รหัสลูกหนี้ (Debtor ID)"
// @Param        Debtor body map[string]interface{} true "ข้อมูลลูกหนี้ที่ต้องการแก้ไข"
// @Success      200  {object} map[string]interface{}
// @Failure      400  {object} map[string]string
// @Failure      404  {object} map[string]string
// @Router       /api/debtors/{id} [put]
func UpdateDebtor(c *gin.Context) {
	// 1. รับ ID ของลูกหนี้จาก URL Parameter
	debtorID := c.Param("id")

	// 2. ค้นหาลูกหนี้เดิมใน Database ว่ามีอยู่จริงไหม
	var debtor models.Debtor
	if err := database.DB.First(&debtor, debtorID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบข้อมูลลูกหนี้รหัสนี้"})
		return
	}

	// 3. รับข้อมูลที่ส่งมาแบบ Map (เพื่อทำ Partial Update)
	var inputMap map[string]interface{}
	if err := c.ShouldBindJSON(&inputMap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "รูปแบบข้อมูลไม่ถูกต้อง: " + err.Error()})
		return
	}

	// 4. กรองข้อมูล (White-list) เฉพาะฟิลด์ที่อนุญาตให้แก้ไข
	updateData := make(map[string]interface{})
	allowedFields := []string{
		"name", 
		"phone", 
		"address", 
		"img_debtor", 
		"credit_limit", 
		// ⚠️ ไม่ใส่ "current_debt", "shop_id", "debtor_id" เพื่อความปลอดภัยของระบบ
	}

	for _, field := range allowedFields {
		if val, exists := inputMap[field]; exists {
			updateData[field] = val
		}
	}

	// ถ้าไม่ได้ส่งอะไรมาเปลี่ยนเลย ให้ตอบกลับไปเลยเพื่อประหยัดการทำงานของ DB
	if len(updateData) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "ไม่มีข้อมูลเปลี่ยนแปลง",
			"data":    debtor,
		})
		return
	}

	// 5. สั่งอัปเดตลง Database
	if err := database.DB.Model(&debtor).Updates(updateData).Error; err != nil {
		// เช็คกรณีแก้เบอร์โทร แล้วเผลอไปซ้ำกับลูกหนี้คนอื่นในร้าน (MySQL Error 1062)
		if strings.Contains(err.Error(), "1062") || strings.Contains(err.Error(), "Duplicate entry") {
			c.JSON(http.StatusConflict, gin.H{"error": "เบอร์โทรศัพท์นี้มีในระบบของร้านท่านแล้ว"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถอัปเดตข้อมูลได้: " + err.Error()})
		return
	}

	// 6. ดึงข้อมูลล่าสุดกลับไปแสดงผล
	var updatedDebtor models.Debtor
	database.DB.First(&updatedDebtor, debtorID)

	c.JSON(http.StatusOK, gin.H{
		"message": "แก้ไขข้อมูลลูกหนี้สำเร็จ",
		"data":    updatedDebtor,
	})
}
