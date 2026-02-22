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
func GetDebtorByAll (c *gin.Context) {
	shopID := c.Query("shop_id")

	if  shopID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาระบุ shop_id"})
		return
	}

	var debtors []models.Debtor

	result := database.DB.Where("shop_id = ?", shopID).Find(&debtors)

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

/*
func GetDebtorHistory(db *sql.DB, debtorID int) (*DebtorDetailResponse, error) {
	var resp DebtorDetailResponse
	var creditLimit sql.NullFloat64 // จัดการกรณี credit_limit เป็น NULL

	// ==========================================
	// 1. ดึงข้อมูล Profile ลูกหนี้
	// ==========================================
	err := db.QueryRow(`
		SELECT debtor_id, name, phone, address, current_debt, credit_limit 
		FROM debtors 
		WHERE debtor_id = ?`, debtorID).
		Scan(&resp.DebtorID, &resp.Name, &resp.Phone, &resp.Address, &resp.CurrentDebt, &creditLimit)

	if err != nil {
		return nil, err // ไม่เจอลูกหนี้ หรือ Database error
	}

	if creditLimit.Valid {
		resp.CreditLimit = creditLimit.Float64
		resp.CreditRemain = resp.CreditLimit - resp.CurrentDebt
	}

	// ==========================================
	// 2. ดึงข้อมูลประวัติบิล (sales) และ สินค้า (sale_items + products)
	// ==========================================
	// ใช้ LEFT JOIN เพื่อให้ดึงบิลออกมาได้แม้จะไม่มี item (ป้องกัน data แหว่ง)
	query := `
		SELECT 
			s.sale_id, s.created_at, s.net_price, s.pay,
			si.amount, si.total_price,
			p.name AS product_name
		FROM sales s
		JOIN sale_items si ON s.sale_id = si.sale_id
		JOIN products p ON si.product_id = p.product_id
		WHERE s.debtor_id = ? 
		  AND s.net_price > s.pay  -- ดึงเฉพาะบิลที่ยอดซื้อมากกว่ายอดที่จ่าย (คือบิลที่ติดหนี้)
		ORDER BY s.created_at DESC
	`

	rows, err := db.Query(query, debtorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// ใช้ Map เพื่อจัดกลุ่ม Item เข้าไปในแต่ละบิล
	billsMap := make(map[int]*DebtBill)
	var billOrder []int // เก็บลำดับ sale_id เพื่อรักษา order ตอนแปลงกลับเป็น Slice

	for rows.Next() {
		var (
			saleID      int
			createdAt   time.Time
			netPrice    float64
			pay         float64
			amount      int
			totalPrice  float64
			productName string
		)

		err := rows.Scan(&saleID, &createdAt, &netPrice, &pay, &amount, &totalPrice, &productName)
		if err != nil {
			return nil, err
		}

		// ถ้ายังไม่มีบิลนี้ใน map ให้สร้างใหม่
		if _, exists := billsMap[saleID]; !exists {
			billsMap[saleID] = &DebtBill{
				SaleID:    saleID,
				CreatedAt: createdAt.Format("02 Jan 2006 15:04"), // Format วันที่ตามต้องการ 
				NetPrice:  netPrice,
				Paid:      pay,
				Remaining: netPrice - pay,
				Items:     []DebtItem{},
			}
			billOrder = append(billOrder, saleID)
		}

		// นำสินค้าใส่เข้าไปในบิลนั้นๆ
		billsMap[saleID].Items = append(billsMap[saleID].Items, DebtItem{
			ProductName: productName,
			Amount:      amount,
			TotalPrice:  totalPrice,
		})
	}

	// แปลง Map กลับเป็น Slice (Array) เพื่อใส่ใน Response
	for _, id := range billOrder {
		resp.Histories = append(resp.Histories, *billsMap[id])
	}

	return &resp, nil
}*/