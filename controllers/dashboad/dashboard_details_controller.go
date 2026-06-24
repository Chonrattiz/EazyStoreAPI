package controller

import (
	"EazyStoreAPI/database"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TransactionDetail struct
type TransactionDetail struct {
	SaleID        int     `json:"sale_id"`
	NetPrice      float64 `json:"net_price"`
	Pay           float64 `json:"pay"`
	PaymentMethod string  `json:"payment_method"`
	CreatedAt     string  `json:"created_at"`
	CreatedTime   *string `json:"created_time"`
}

// GetTransactionsDetail ดึงรายการบิลในช่วงเวลา
func GetTransactionsDetail(c *gin.Context) {
	shopID := c.Query("shop_id")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if shopID == "" || startDate == "" || endDate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาส่ง shop_id, start_date และ end_date"})
		return
	}

	var transactions []TransactionDetail
	database.DB.Table("sales").
		Select("sale_id, net_price, pay, payment_method, DATE_FORMAT(created_at, '%Y-%m-%d') as created_at, created_time").
		Where("shop_id = ? AND created_at >= ? AND created_at <= ?", shopID, startDate, endDate).
		Order("created_at DESC, created_time DESC").
		Scan(&transactions)

	c.JSON(http.StatusOK, gin.H{"transactions": transactions})
}

// ProductSalesDetail struct
type ProductSalesDetail struct {
	ProductID   int     `json:"product_id"`
	ProductName string  `json:"product_name"`
	ImgProduct  string  `json:"img_product"`
	TotalQty    int     `json:"total_qty"`
	TotalSales  float64 `json:"total_sales"`
	TotalCost   float64 `json:"total_cost"`
	Profit      float64 `json:"profit"`
}

// GetProductSalesDetail ดึงข้อมูลยอดขายแยกตามสินค้าในช่วงเวลา
func GetProductSalesDetail(c *gin.Context) {
	shopID := c.Query("shop_id")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if shopID == "" || startDate == "" || endDate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาส่ง shop_id, start_date และ end_date"})
		return
	}

	var details []ProductSalesDetail
	database.DB.Table("sale_items").
		Select(`
			products.product_id, 
			products.name as product_name, 
			products.img_product,
			SUM(sale_items.amount) as total_qty, 
			SUM(sale_items.total_price) as total_sales, 
			SUM(sale_items.amount * products.cost_price) as total_cost,
			SUM(sale_items.total_price) - SUM(sale_items.amount * products.cost_price) as profit
		`).
		Joins("JOIN sales ON sales.sale_id = sale_items.sale_id").
		Joins("JOIN products ON products.product_id = sale_items.product_id").
		Where("sales.shop_id = ? AND sales.created_at >= ? AND sales.created_at <= ?", shopID, startDate, endDate).
		Group("products.product_id, products.name, products.img_product").
		Order("total_sales DESC").
		Scan(&details)

	c.JSON(http.StatusOK, gin.H{"product_details": details})
}

// SaleDetailItem โครงสร้างรายการสินค้าในบิลขาย
type SaleDetailItem struct {
	ProductID   int     `json:"product_id"`
	ProductName string  `json:"product_name"`
	ImgProduct  string  `json:"img_product"`
	Qty         int     `json:"qty"`
	UnitPrice   float64 `json:"unit_price"`
	CostPrice   float64 `json:"cost_price"`
	Subtotal    float64 `json:"subtotal"`
}

// SaleDetailResponse โครงสร้างข้อมูลบิลขายและรายการสินค้าทั้งหมด
type SaleDetailResponse struct {
	SaleID        int              `json:"sale_id"`
	CreatedAt     string           `json:"created_at"`
	CreatedTime   string           `json:"created_time"`
	PaymentMethod string           `json:"payment_method"`
	NetPrice      float64          `json:"net_price"`
	Pay           float64          `json:"pay"`
	Change        float64          `json:"change"`
	Items         []SaleDetailItem `json:"items"`
}

// GetSaleItems ดึงข้อมูลรายละเอียดบิลและรายการสินค้าในบิลนั้นๆ
func GetSaleItems(c *gin.Context) {
	shopID := c.Query("shop_id")
	saleID := c.Query("sale_id")

	if shopID == "" || saleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาส่ง shop_id และ sale_id"})
		return
	}

	// ดึงข้อมูลหลักของบิลขาย
	var sale struct {
		SaleID        int     `gorm:"column:sale_id"`
		CreatedAt     string  `gorm:"column:created_at"`
		CreatedTime   *string `gorm:"column:created_time"`
		PaymentMethod string  `gorm:"column:payment_method"`
		NetPrice      float64 `gorm:"column:net_price"`
		Pay           float64 `gorm:"column:pay"`
	}

	err := database.DB.Table("sales").
		Select("sale_id, DATE_FORMAT(created_at, '%Y-%m-%d') as created_at, created_time, payment_method, net_price, pay").
		Where("shop_id = ? AND sale_id = ?", shopID, saleID).
		First(&sale).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ไม่พบข้อมูลบิลขาย"})
		return
	}

	// คำนวณเงินทอนแบบไดนามิก
	change := sale.Pay - sale.NetPrice
	if change < 0 {
		change = 0
	}

	// ดึงรายละเอียดรายการสินค้าในบิลขาย
	var items []SaleDetailItem
	err = database.DB.Table("sale_items").
		Select(`
			sale_items.product_id, 
			products.name as product_name, 
			products.img_product, 
			sale_items.amount as qty, 
			sale_items.price_per_unit as unit_price, 
			products.cost_price as cost_price, 
			sale_items.total_price as subtotal
		`).
		Joins("LEFT JOIN products ON products.product_id = sale_items.product_id").
		Where("sale_items.sale_id = ?", sale.SaleID).
		Scan(&items).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ไม่สามารถดึงข้อมูลรายการสินค้าได้"})
		return
	}

	createdTimeStr := ""
	if sale.CreatedTime != nil {
		createdTimeStr = *sale.CreatedTime
	}

	// จัดรูป Response ส่งกลับไปยัง Flutter
	response := SaleDetailResponse{
		SaleID:        sale.SaleID,
		CreatedAt:     sale.CreatedAt,
		CreatedTime:   createdTimeStr,
		PaymentMethod: sale.PaymentMethod,
		NetPrice:      sale.NetPrice,
		Pay:           sale.Pay,
		Change:        change,
		Items:         items,
	}

	c.JSON(http.StatusOK, response)
}

