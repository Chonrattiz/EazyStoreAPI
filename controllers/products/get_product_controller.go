package controllers

import (
	"EazyStoreAPI/database"

	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"EazyStoreAPI/middleware"
	"EazyStoreAPI/models"

	"github.com/gin-gonic/gin"
)

// thaiSortKey สลับตำแหน่งสระนำ (เ แ โ ใ ไ) กับตัวพยัญชนะถัดไป เพราะ MySQL
// (utf8mb4) ไม่มี collation ภาษาไทยที่เรียงตามพจนานุกรมจริง การ ORDER BY name
// ตรงๆ จะเรียงตาม code point ทำให้คำที่ขึ้นต้นด้วยสระนำเหล่านี้ไปอยู่ท้ายสุด
// เสมอ ทั้งที่ควรเรียงถัดจากพยัญชนะที่ตามมา (เช่น "เครื่องดื่ม" ควรอยู่แถว ค)
func thaiSortKey(input string) string {
	leadingVowels := map[rune]bool{'เ': true, 'แ': true, 'โ': true, 'ใ': true, 'ไ': true}
	runes := []rune(input)
	var sb strings.Builder
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		if leadingVowels[ch] && i+1 < len(runes) {
			sb.WriteRune(runes[i+1])
			sb.WriteRune(ch)
			i++
		} else {
			sb.WriteRune(ch)
		}
	}
	return sb.String()
}

// GetCategories ดึงหมวดหมู่ของร้านที่ระบุ (ต้องส่ง shop_id มา)
func GetCategories(c *gin.Context) {
	shopID, ok := middleware.RequireShopIDQuery(c)
	if !ok {
		return
	}

	var categories []models.Category

	if err := database.DB.Where("shop_id = ? AND status = ?", shopID, true).Order("category_id ASC").Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถดึงข้อมูลหมวดหมู่ได้"})
		return
	}

	c.JSON(http.StatusOK, categories)
}

// GetProductsByShop godoc
// @Summary      ดึงรายการสินค้าทั้งหมดของร้านค้า
// @Description  ดึงข้อมูลสินค้าทั้งหมดที่ผูกกับ shop_id โดยเรียงจากใหม่ไปเก่า
// @Tags         Product
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        shop_id   query     int  true  "รหัสร้านค้า (Shop ID)"
// @Param         page      query     int    false  "หน้าปัจจุบัน (เริ่มที่ 1)"
// @Param         limit     query     int    false  "จำนวนรายการต่อหน้า"
// @Param         search      query     String    false  "ค้นหา"
// @Param         category_id     query     int    false  "หมวดหมู่"
// @Success      200  {array}   models.Product
// @Failure      400  {object}  map[string]string "กรุณาระบุ shop_id"
// @Failure      500  {object}  map[string]string "ไม่สามารถดึงข้อมูลสินค้าได้"
// @Router       /api/products [get]
func GetProductsByShop(c *gin.Context) {
	shopID, ok := middleware.RequireShopIDQuery(c)
	if !ok {
		return
	}

	search := c.Query("search")
	categoryID := c.Query("category_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1")) // เปลี่ยน default เป็น 1
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// ✨ เพิ่มการรับค่าการเรียงลำดับจาก App (เช่น asc หรือ desc)
	sortOrder := c.DefaultQuery("sort", "desc")

	var products []models.Product
	var totalItems int64

	// 1. สร้าง Query พื้นฐาน
	query := database.DB.Model(&models.Product{}).Preload("Category").Where("shop_id = ?", shopID)

	// 2. Filter: ค้นหา
	if search != "" {
		query = query.Where("name LIKE ? OR barcode LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// 3. Filter: หมวดหมู่ (เช็คให้ชัวร์ว่าไม่ใช่ "0" หรือว่าง)
	if categoryID != "" && categoryID != "0" {
		query = query.Where("category_id = ?", categoryID)
	}

	// 4. นับจำนวนรวมหลังจาก Filter แล้ว (เพื่อให้ Total Pages ถูกต้อง)
	query.Count(&totalItems)

	// 5. ✨ Logic การเรียงลำดับ: ต้องเรียง "ก่อน" ทำ Limit/Offset
	// รองรับ name_asc / name_desc / stock_asc / stock_desc (รูปแบบที่ App ส่งมาปัจจุบัน)
	// และคง asc/desc แบบเดิมไว้เพื่อ backward compatibility (หมายถึงสต็อก)
	offset := (page - 1) * limit

	switch sortOrder {
	case "name_asc", "name_desc":
		// MySQL (utf8mb4) ไม่มี collation ภาษาไทยที่เรียงตามพจนานุกรมจริง
		// ORDER BY name ตรงๆ จะเรียงผิด จึงต้องดึงข้อมูลที่ผ่าน filter มา
		// ทั้งหมดก่อน แล้วเรียงเองด้วย thaiSortKey จากนั้นค่อยตัดหน้าด้วยมือ
		// เพื่อให้ผลลัพธ์เรียงถูกต่อเนื่องกันทุกหน้า ไม่ใช่แค่ในหน้าเดียว
		if err := query.Find(&products).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถดึงข้อมูลสินค้าได้"})
			return
		}
		sort.SliceStable(products, func(i, j int) bool {
			ki, kj := thaiSortKey(products[i].Name), thaiSortKey(products[j].Name)
			if sortOrder == "name_desc" {
				return kj < ki
			}
			return ki < kj
		})
		start := offset
		if start > len(products) {
			start = len(products)
		}
		end := start + limit
		if end > len(products) {
			end = len(products)
		}
		products = products[start:end]
	case "stock_asc", "asc":
		if err := query.Order("stock ASC").Limit(limit).Offset(offset).Find(&products).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถดึงข้อมูลสินค้าได้"})
			return
		}
	default: // "stock_desc", "desc" หรือค่าอื่นที่ไม่รู้จัก
		if err := query.Order("stock DESC").Limit(limit).Offset(offset).Find(&products).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถดึงข้อมูลสินค้าได้"})
			return
		}
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))

	c.JSON(http.StatusOK, gin.H{
		"items":        products,
		"total_items":  totalItems,
		"total_pages":  totalPages,
		"current_page": page,
	})
}

// GetProductBySearch godoc
// @Summary      ค้นหาสินค้า (Search Product)
// @Description  ค้นหาสินค้าด้วย Keyword เฉพาะในร้านที่ระบุ (ป้องกันการเจอของร้านอื่น)
// @Tags         Product
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        keyword    query    string   true   "คำค้นหา (Barcode, รหัส, ชื่อ)"
// @Param        shop_id    query    int      true   "รหัสร้านค้า (Shop ID)"
// @Success      200  {object}  models.Product
// @Failure      400  {object}  map[string]string "Bad Request"
// @Failure      404  {object}  map[string]string "Product not found"
// @Router       /api/product/search [get]
func GetProductBySearch(c *gin.Context) {
	// 1. รับค่า shop_id พร้อมตรวจสิทธิ์ว่าเป็นร้านของ user ที่ล็อกอินอยู่
	shopID, ok := middleware.RequireShopIDQuery(c)
	if !ok {
		return
	}

	// 2. ตรวจสอบว่าส่ง keyword มาไหม
	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาระบุคำค้นหา และ รหัสร้านค้า"})
		return
	}

	var product models.Product

	// 3. ค้นหาโดยระบุ shop_id ด้วย
	// SQL: SELECT * FROM products WHERE shop_id = ? AND (product_code = ? OR barcode = ? OR name LIKE ?) LIMIT 1
	result := database.DB.Preload("Category").
		Where("shop_id = ?", shopID). // ✅ ล็อคให้หาแค่ในร้านนี้เท่านั้น!
		Where("product_code = ? OR barcode = ? OR name LIKE ?", keyword, keyword, "%"+keyword+"%").
		First(&product)

	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบสินค้าในร้านนี้"})
		return
	}

	c.JSON(http.StatusOK, product)
}

// GetNullBarcode godoc
// @Summary      ดูสินค้าไม่มีบาร์โค้ด
// @Description  ดึงรายการสินค้าที่ไม่มีบาร์โค้ดแยกตามร้านค้า
// @Tags         Product
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        shop_id   query     int    true  "รหัสร้านค้า (Shop ID)"
// @Param        category_id  query    int     false "รหัสหมวดหมู่สินค้า (Category ID)"
// @Success      200  {array}   models.Product
// @Failure      404  {object}  map[string]string "Product not found"
// @Failure      500  {object}  map[string]string "Internal Error"
// @Router       /api/getNullBarcode [get]
func GetNullBarcode(c *gin.Context) {
	shopID, ok := middleware.RequireShopIDQuery(c)
	if !ok {
		return
	}

	categoryID := c.Query("category_id")

	var products []models.Product
	query := database.DB.Where("shop_id = ? AND barcode IS NULL", shopID)

	// ถ้ามีการส่ง category_id มา ให้เพิ่มเงื่อนไขการค้นหา
	if categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}

	err := query.Order("category_id ASC").Find(&products).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ดึงข้อมูลล้มเหลว"})
		return
	}

	c.JSON(http.StatusOK, products)
}
