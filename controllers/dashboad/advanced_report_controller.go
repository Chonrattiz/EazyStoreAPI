package controller

import (
	"EazyStoreAPI/database"
	"EazyStoreAPI/middleware"
	"EazyStoreAPI/models"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func GetAdvancedReport(c *gin.Context) {
	shopID, ok := middleware.RequireShopIDQuery(c)
	if !ok {
		return
	}

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if startDate == "" || endDate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาส่ง shop_id, start_date และ end_date ให้ครบถ้วน"})
		return
	}

	// 1. กราฟยอดขาย (Sales Chart)
	var salesChart []models.ChartItem
	if startDate == endDate {
		// รายวัน (วันนี้): Group ตามชั่วโมง 00:00 - 23:00
		var rawSalesChart []models.ChartItem
		database.DB.Table("sales").
			Select("DATE_FORMAT(CONCAT(DATE(created_at), ' ', COALESCE(created_time, '00:00:00')), '%H:00') as date, COALESCE(SUM(net_price), 0) as total_sales").
			Where("shop_id = ? AND DATE(created_at) = ?", shopID, startDate).
			Group("DATE_FORMAT(CONCAT(DATE(created_at), ' ', COALESCE(created_time, '00:00:00')), '%H:00')").
			Order("date ASC").
			Scan(&rawSalesChart)

		salesMap := make(map[string]float64)
		for _, item := range rawSalesChart {
			salesMap[item.Date] = item.TotalSales
		}

		for hour := 0; hour < 24; hour++ {
			hourStr := fmt.Sprintf("%02d:00", hour)
			dateStr := fmt.Sprintf("%sT%s:00", startDate, hourStr)
			salesChart = append(salesChart, models.ChartItem{
				Date:       dateStr,
				TotalSales: salesMap[hourStr],
			})
		}
	} else {
		// รายช่วงเวลา (เดือนนี้ / ปีนี้): Group ตามวัน
		var rawSalesChart []models.ChartItem
		database.DB.Table("sales").
			Select("DATE(created_at) as date, COALESCE(SUM(net_price), 0) as total_sales").
			Where("shop_id = ? AND DATE(created_at) >= ? AND DATE(created_at) <= ?", shopID, startDate, endDate).
			Group("DATE(created_at)").
			Order("date ASC").
			Scan(&rawSalesChart)

		salesMap := make(map[string]float64)
		for _, item := range rawSalesChart {
			dateOnly := item.Date
			if len(dateOnly) >= 10 {
				dateOnly = dateOnly[:10]
			}
			salesMap[dateOnly] = item.TotalSales
		}

		start, err1 := time.Parse("2006-01-02", startDate)
		end, err2 := time.Parse("2006-01-02", endDate)
		if err1 == nil && err2 == nil {
			for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
				dateStr := d.Format("2006-01-02")
				salesChart = append(salesChart, models.ChartItem{
					Date:       dateStr,
					TotalSales: salesMap[dateStr],
				})
			}
		} else {
			salesChart = rawSalesChart
		}
	}

	// 2. สัดส่วนวิธีการชำระเงิน (Payment Methods Breakdown)
	// สิ่งที่ควรทราบ: แยกยอดรับจริง (หักเงินทอน) ตามประเภท "เงินสด" และ "โอนจ่าย"
	var paymentStats models.PaymentStats
	database.DB.Table("sales").
		Select(`
			COALESCE(SUM(CASE WHEN payment_method = 'จ่ายเงินสด' THEN (CASE WHEN pay >= net_price THEN net_price ELSE pay END) ELSE 0 END), 0) as paid_cash,
			COALESCE(SUM(CASE WHEN payment_method = 'โอนจ่าย' THEN (CASE WHEN pay >= net_price THEN net_price ELSE pay END) ELSE 0 END), 0) as paid_transfer
		`).
		Where("shop_id = ? AND DATE(created_at) >= ? AND DATE(created_at) <= ?", shopID, startDate, endDate).
		Scan(&paymentStats)

	// สิ่งที่ควรทราบ: ยอดหนี้ (debt_amount) ต้องเอาไปหักลบกับประวัติการชำระหนี้ (debt_payments)
	// ด้วยหลักการเข้าก่อนออกก่อน (FIFO waterfall) เพื่อให้ได้ยอดหนี้คงค้างที่แท้จริง
	// ไม่ใช่ดึงแค่ net_price - pay เพราะเราไม่อัปเดตค่า pay ย้อนหลังเมื่อมีการชำระหนี้เพิ่ม
	database.DB.Raw(`
		WITH sale_debts AS (
			SELECT
				s.sale_id, s.debtor_id, s.created_at,
				(s.net_price - s.pay) AS original_debt,
				SUM(s.net_price - s.pay) OVER (
					PARTITION BY s.debtor_id ORDER BY s.created_at, s.sale_id
				) AS cumulative_debt
			FROM sales s
			WHERE s.shop_id = ?
				AND s.debtor_id IS NOT NULL
				AND (s.pay < s.net_price OR s.payment_method = 'ค้างชำระ')
		),
		debtor_paid AS (
			SELECT dp.debtor_id, SUM(dp.amount_paid) AS total_paid
			FROM debt_payments dp
			JOIN debtors d ON d.debtor_id = dp.debtor_id
			WHERE d.shop_id = ?
			GROUP BY dp.debtor_id
		)
		SELECT COALESCE(SUM(
			GREATEST(0, LEAST(sd.original_debt, sd.cumulative_debt - COALESCE(p.total_paid, 0)))
		), 0) AS debt_amount
		FROM sale_debts sd
		LEFT JOIN debtor_paid p ON p.debtor_id = sd.debtor_id
		WHERE DATE(sd.created_at) >= ? AND DATE(sd.created_at) <= ?
	`, shopID, shopID, startDate, endDate).Scan(&paymentStats.DebtAmount)

	// 3. สินค้าขายดี 5 อันดับแรก (Top 5 Products)
	var topProducts []models.TopProduct
	database.DB.Table("sale_items").
		Select("products.name as product_name, SUM(sale_items.amount) as total_qty, SUM(sale_items.total_price) as total_sales").
		Joins("JOIN sales ON sales.sale_id = sale_items.sale_id").
		Joins("JOIN products ON products.product_id = sale_items.product_id").
		Where("sales.shop_id = ? AND DATE(sales.created_at) >= ? AND DATE(sales.created_at) <= ?", shopID, startDate, endDate).
		Group("products.product_id, products.name").
		Order("total_qty DESC").
		Limit(5).
		Scan(&topProducts)

	// 4. สรุปภาพรวมหนี้สิน (Debt Summary)
	var debtSummary models.DebtSummary
	// สิ่งที่ควรทราบ: คำนวณยอดหนี้คงค้างทั้งหมด (Total Outstanding)
	database.DB.Table("debtors").
		Select("COALESCE(SUM(current_debt), 0) as total_outstanding").
		Where("shop_id = ?", shopID).
		Scan(&debtSummary)

	// ยอดหนี้ที่เก็บได้ในช่วงเวลานี้ (ตามตัวกรองวันที่ startDate - endDate)
	database.DB.Table("debt_payments").
		Select("COALESCE(SUM(amount_paid), 0) as collected_this_month").
		Joins("JOIN debtors ON debtors.debtor_id = debt_payments.debtor_id").
		Where("debtors.shop_id = ? AND DATE(payment_date) >= ? AND DATE(payment_date) <= ?", shopID, startDate, endDate).
		Scan(&debtSummary.CollectedThisMonth)

	// 5. รายงานอายุหนี้ (Aging Report)
	// สิ่งที่ควรทราบ: ส่วนนี้คือหัวใจสำคัญของการบริหารลูกหนี้!
	// - การคำนวณอายุหนี้จะอิงตาม "ลูกหนี้" (Debtor) ไม่ใช่แยกตาม "บิล" (Sale)
	// - เมื่อลูกหนี้มาชำระเงิน ระบบจะใช้วิธีนำเงินไปตัดหนี้บิลที่เก่าที่สุดก่อน (FIFO Waterfall)
	// - "อายุหนี้" จะถูกรีเซ็ตนับจากวันที่ของ "บิลที่เก่าที่สุดที่ยังจ่ายไม่หมด" หรือ "วันที่ชำระเงินครั้งล่าสุด" (เอาอันที่ใหม่กว่า)
	// - การนับอายุหนี้จะคำนวณเทียบกับ "วันปัจจุบัน (as of today)" เสมอ เพื่อให้เห็นสถานะความเสี่ยงล่าสุด
	//   โดยไม่สนใจว่าผู้ใช้จะตั้งตัวกรองช่วงเวลาในรายงานเป็นเดือนไหนก็ตาม (ต่างจากกราฟยอดขายด้านบนที่เปลี่ยนตามตัวกรอง)
	asOfDate := time.Now().Format("2006-01-02")
	var aging models.AgingStats
	database.DB.Raw(`
		WITH sale_debts AS (  
			SELECT
				s.sale_id, s.debtor_id, s.created_at,  
				(s.net_price - s.pay) AS original_debt,
				SUM(s.net_price - s.pay) OVER (
					PARTITION BY s.debtor_id ORDER BY s.created_at, s.sale_id
				) AS cumulative_debt
			FROM sales s
			WHERE s.shop_id = ?
				AND s.debtor_id IS NOT NULL
				AND s.created_at <= ?
				AND (s.pay < s.net_price OR s.payment_method = 'ค้างชำระ')
		),
		debtor_paid AS (
			SELECT dp.debtor_id, SUM(dp.amount_paid) AS total_paid
			FROM debt_payments dp
			JOIN debtors d ON d.debtor_id = dp.debtor_id
			WHERE d.shop_id = ? AND DATE(dp.payment_date) <= ?
			GROUP BY dp.debtor_id
		),
		remaining_per_sale AS (
			SELECT
				sd.debtor_id, sd.created_at,
				GREATEST(0, LEAST(sd.original_debt, sd.cumulative_debt - COALESCE(p.total_paid, 0))) AS remaining
			FROM sale_debts sd
			LEFT JOIN debtor_paid p ON p.debtor_id = sd.debtor_id
		),
		debtor_totals AS (
			SELECT
				r.debtor_id,
				SUM(r.remaining) AS amount_owed,
				MIN(CASE WHEN r.remaining > 0 THEN r.created_at END) AS oldest_open_sale_date
			FROM remaining_per_sale r
			GROUP BY r.debtor_id
			HAVING SUM(r.remaining) > 0
		),
		debt_events AS (
			SELECT s.debtor_id, s.created_at AS event_date, 1 AS event_order, (s.net_price - s.pay) AS delta
			FROM sales s
			WHERE s.shop_id = ? AND s.debtor_id IS NOT NULL AND s.created_at <= ?
				AND (s.pay < s.net_price OR s.payment_method = 'ค้างชำระ')
			UNION ALL
			SELECT d.debtor_id, DATE(dp.payment_date) AS event_date, 2 AS event_order, -dp.amount_paid AS delta
			FROM debt_payments dp JOIN debtors d ON d.debtor_id = dp.debtor_id
			WHERE d.shop_id = ? AND DATE(dp.payment_date) <= ?
		),
		running_balance AS (
			SELECT debtor_id, event_date,
				SUM(delta) OVER (PARTITION BY debtor_id ORDER BY event_date, event_order) AS balance
			FROM debt_events
		),
		cycle_start AS (
			SELECT debtor_id, MAX(event_date) AS cycle_start_date
			FROM running_balance
			WHERE balance <= 0
			GROUP BY debtor_id
		),
		debtor_paid_in_cycle AS (
			SELECT dp.debtor_id, MAX(DATE(dp.payment_date)) AS last_payment_date
			FROM debt_payments dp
			JOIN debtors d ON d.debtor_id = dp.debtor_id
			LEFT JOIN cycle_start cs ON cs.debtor_id = dp.debtor_id
			WHERE d.shop_id = ? AND DATE(dp.payment_date) <= ?
				AND DATE(dp.payment_date) >= COALESCE(cs.cycle_start_date, '1900-01-01')
			GROUP BY dp.debtor_id
		),
		debtor_aging AS (
			SELECT
				t.debtor_id,
				t.amount_owed,
				GREATEST(
					COALESCE(p.last_payment_date, t.oldest_open_sale_date),
					t.oldest_open_sale_date
				) AS debt_since
			FROM debtor_totals t
			LEFT JOIN debtor_paid_in_cycle p ON p.debtor_id = t.debtor_id
		)
		SELECT
			COALESCE(SUM(CASE WHEN DATEDIFF(?, debt_since) + 1 <= 15 THEN amount_owed ELSE 0 END), 0) as safe,
			COALESCE(SUM(CASE WHEN DATEDIFF(?, debt_since) + 1 BETWEEN 16 AND 30 THEN amount_owed ELSE 0 END), 0) as warning,
			COALESCE(SUM(CASE WHEN DATEDIFF(?, debt_since) + 1 > 30 THEN amount_owed ELSE 0 END), 0) as danger
		FROM debtor_aging
	`, shopID, asOfDate, // sale_debts
		shopID, asOfDate, // debtor_paid
		shopID, asOfDate, 
		shopID, asOfDate, // debt_events
		shopID, asOfDate, // debtor_paid_in_cycle
		asOfDate, asOfDate, asOfDate, // final SELECT
	).Scan(&aging)

	// 6. ลูกหนี้ยอดสูงสุด 5 อันดับแรก (Top 5 Debtors)
	var topDebtors []models.TopDebtor
	database.DB.Table("debtors").
		Select("debtor_id, name, current_debt").
		Where("shop_id = ? AND current_debt > 0", shopID).
		Order("current_debt DESC").
		Limit(5).
		Scan(&topDebtors)

	// 7. สรุปความเคลื่อนไหวหนี้ (Debt Collection Statement)
	// สิ่งที่ควรทราบ: เป็นการเปรียบเทียบระหว่าง "หนี้ใหม่ที่เกิดขึ้น" กับ "หนี้เก่าที่เก็บได้" ในช่วงเวลาที่เลือก
	var debtCollection models.DebtCollection
	database.DB.Table("sales").
		Select("COALESCE(SUM(net_price - pay), 0) as new_debt").
		Where("shop_id = ? AND DATE(created_at) >= ? AND DATE(created_at) <= ? AND (pay < net_price OR payment_method = 'ค้างชำระ')", shopID, startDate, endDate).
		Scan(&debtCollection.NewDebt)

	database.DB.Table("debt_payments").
		Select("COALESCE(SUM(amount_paid), 0) as collected_debt").
		Joins("JOIN debtors ON debtors.debtor_id = debt_payments.debtor_id").
		Where("debtors.shop_id = ? AND DATE(payment_date) >= ? AND DATE(payment_date) <= ?", shopID, startDate, endDate).
		Scan(&debtCollection.CollectedDebt)

	// รวมข้อมูลทั้งหมดและส่งกลับเป็น JSON ไปให้แอป Flutter
	// โดยแต่ละตัวแปร จะเอาไปแสดงผลในส่วนต่างๆ ของหน้า UI ดังนี้:
	c.JSON(http.StatusOK, gin.H{
		"sales_chart":     salesChart,
		"payment_methods": paymentStats,
		"top_products":    topProducts,
		"debt_summary":    debtSummary,
		"aging_report":    aging,
		"top_debtors":     topDebtors,
		"debt_collection": debtCollection,
	})
}

// GetAgingReportDetail ดึงรายชื่อลูกหนี้แยกตามกลุ่มอายุหนี้ (Safe/Warning/Danger)
// สิ่งที่ควรทราบ: ฟังก์ชันนี้ใช้ตรรกะคำนวณหนี้แบบ FIFO Waterfall เหมือนกับใน GetAdvancedReport เป๊ะ
// เพื่อให้ผู้ใช้สามารถกดเจาะลึก (Drill down) จากกราฟแท่งอายุหนี้ เข้ามาดูรายชื่อคนติดหนี้แต่ละคนได้
// และยอดรวมของหนี้ในกลุ่ม Safe/Warning/Danger ตรงนี้จะตรงกับกราฟในหน้า Dashboard เสมอ
func GetAgingReportDetail(c *gin.Context) {
	shopID, ok := middleware.RequireShopIDQuery(c)
	if !ok {
		return
	}

	endDate := c.Query("end_date")
	if endDate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "กรุณาส่ง shop_id และ end_date ให้ครบถ้วน"})
		return
	}

	// สิ่งที่ควรทราบ: อายุหนี้คำนวณแบบ Real-time โดยเทียบกับวันปัจจุบัน (as of today) เสมอ
	asOfDate := time.Now().Format("2006-01-02")

	var debtors []models.AgingDebtor
	database.DB.Raw(`
		WITH sale_debts AS (
			SELECT
				s.sale_id, s.debtor_id, s.created_at,
				(s.net_price - s.pay) AS original_debt,
				SUM(s.net_price - s.pay) OVER (
					PARTITION BY s.debtor_id ORDER BY s.created_at, s.sale_id
				) AS cumulative_debt
			FROM sales s
			WHERE s.shop_id = ?
				AND s.debtor_id IS NOT NULL
				AND s.created_at <= ?
				AND (s.pay < s.net_price OR s.payment_method = 'ค้างชำระ')
		),
		debtor_paid AS (
			SELECT dp.debtor_id, SUM(dp.amount_paid) AS total_paid
			FROM debt_payments dp
			JOIN debtors d ON d.debtor_id = dp.debtor_id
			WHERE d.shop_id = ? AND DATE(dp.payment_date) <= ?
			GROUP BY dp.debtor_id
		),
		remaining_per_sale AS (
			SELECT
				sd.debtor_id, sd.created_at,
				GREATEST(0, LEAST(sd.original_debt, sd.cumulative_debt - COALESCE(p.total_paid, 0))) AS remaining
			FROM sale_debts sd
			LEFT JOIN debtor_paid p ON p.debtor_id = sd.debtor_id
		),
		debtor_totals AS (
			SELECT
				r.debtor_id,
				SUM(r.remaining) AS amount_owed,
				MIN(CASE WHEN r.remaining > 0 THEN r.created_at END) AS oldest_open_sale_date
			FROM remaining_per_sale r
			GROUP BY r.debtor_id
			HAVING SUM(r.remaining) > 0
		),
		debt_events AS (
			SELECT s.debtor_id, s.created_at AS event_date, 1 AS event_order, (s.net_price - s.pay) AS delta
			FROM sales s
			WHERE s.shop_id = ? AND s.debtor_id IS NOT NULL AND s.created_at <= ?
				AND (s.pay < s.net_price OR s.payment_method = 'ค้างชำระ')
			UNION ALL
			SELECT d.debtor_id, DATE(dp.payment_date) AS event_date, 2 AS event_order, -dp.amount_paid AS delta
			FROM debt_payments dp JOIN debtors d ON d.debtor_id = dp.debtor_id
			WHERE d.shop_id = ? AND DATE(dp.payment_date) <= ?
		),
		running_balance AS (
			SELECT debtor_id, event_date,
				SUM(delta) OVER (PARTITION BY debtor_id ORDER BY event_date, event_order) AS balance
			FROM debt_events
		),
		cycle_start AS (
			SELECT debtor_id, MAX(event_date) AS cycle_start_date
			FROM running_balance
			WHERE balance <= 0
			GROUP BY debtor_id
		),
		debtor_paid_in_cycle AS (
			SELECT dp.debtor_id, MAX(DATE(dp.payment_date)) AS last_payment_date
			FROM debt_payments dp
			JOIN debtors d ON d.debtor_id = dp.debtor_id
			LEFT JOIN cycle_start cs ON cs.debtor_id = dp.debtor_id
			WHERE d.shop_id = ? AND DATE(dp.payment_date) <= ?
				AND DATE(dp.payment_date) >= COALESCE(cs.cycle_start_date, '1900-01-01')
			GROUP BY dp.debtor_id
		),
		debtor_aging AS (
			SELECT
				t.debtor_id,
				t.amount_owed,
				GREATEST(
					COALESCE(p.last_payment_date, t.oldest_open_sale_date),
					t.oldest_open_sale_date
				) AS debt_since
			FROM debtor_totals t
			LEFT JOIN debtor_paid_in_cycle p ON p.debtor_id = t.debtor_id
		)
		SELECT
			a.debtor_id, d.name, d.phone, d.img_debtor,
			a.debt_since,
			a.amount_owed,
			DATEDIFF(?, a.debt_since) + 1 AS days_overdue,
			CASE
				WHEN DATEDIFF(?, a.debt_since) + 1 <= 15 THEN 'safe'
				WHEN DATEDIFF(?, a.debt_since) + 1 BETWEEN 16 AND 30 THEN 'warning'
				ELSE 'danger'
			END AS bucket
		FROM debtor_aging a
		JOIN debtors d ON d.debtor_id = a.debtor_id
		ORDER BY days_overdue DESC, d.name
	`, shopID, asOfDate, // sale_debts
		shopID, asOfDate, // debtor_paid
		shopID, asOfDate, shopID, asOfDate, // debt_events
		shopID, asOfDate, // debtor_paid_in_cycle
		asOfDate, asOfDate, asOfDate, // final SELECT
	).Scan(&debtors)

	safe := []models.AgingDebtor{}
	warning := []models.AgingDebtor{}
	danger := []models.AgingDebtor{}
	for _, b := range debtors {
		switch b.Bucket {
		case "safe":
			safe = append(safe, b)
		case "warning":
			warning = append(warning, b)
		default:
			danger = append(danger, b)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"aging_report_detail": gin.H{
			"safe":    safe,
			"warning": warning,
			"danger":  danger,
		},
	})
}
