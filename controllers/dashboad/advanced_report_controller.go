package controller

import (
	"EazyStoreAPI/database"
	"EazyStoreAPI/middleware"
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

	// 1. Sales Chart (Hourly if 1 day, Daily otherwise)
	type ChartItem struct {
		Date       string  `json:"date"`
		TotalSales float64 `json:"total_sales"`
	}
	var salesChart []ChartItem
	if startDate == endDate {
		// Group by Hour
		database.DB.Table("sales").
			Select("HOUR(created_time) as date, COALESCE(SUM(net_price), 0) as total_sales").
			Where("shop_id = ? AND created_at = ?", shopID, startDate).
			Group("HOUR(created_time)").
			Order("date ASC").
			Scan(&salesChart)
	} else {
		var rawSalesChart []ChartItem
		database.DB.Table("sales").
			Select("DATE(created_at) as date, COALESCE(SUM(net_price), 0) as total_sales").
			Where("shop_id = ? AND created_at >= ? AND created_at <= ?", shopID, startDate, endDate).
			Group("DATE(created_at)").
			Order("date ASC").
			Scan(&rawSalesChart)

		// Create a map for quick lookup
		salesMap := make(map[string]float64)
		for _, item := range rawSalesChart {
			// Extract just the "YYYY-MM-DD" part in case it has time part appended
			dateOnly := item.Date
			if len(dateOnly) >= 10 {
				dateOnly = dateOnly[:10]
			}
			salesMap[dateOnly] = item.TotalSales
		}

		// Generate all dates between startDate and endDate
		start, err1 := time.Parse("2006-01-02", startDate)
		end, err2 := time.Parse("2006-01-02", endDate)
		if err1 == nil && err2 == nil {
			for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
				dateStr := d.Format("2006-01-02")
				salesChart = append(salesChart, ChartItem{
					Date:       dateStr,
					TotalSales: salesMap[dateStr],
				})
			}
		} else {
			salesChart = rawSalesChart
		}
	}

	// 1.5 Summary Stats (Transactions, Net Sales, Average)
	var summaryStats struct {
		TotalTransactions int     `json:"total_transactions"`
		TotalSales        float64 `json:"total_sales"`
		AverageSales      float64 `json:"average_sales"`
	}
	database.DB.Table("sales").
		Select(`
			COUNT(sale_id) as total_transactions,
			COALESCE(SUM(net_price), 0) as total_sales,
			COALESCE(AVG(net_price), 0) as average_sales
		`).
		Where("shop_id = ? AND created_at >= ? AND created_at <= ?", shopID, startDate, endDate).
		Scan(&summaryStats)

	// 2. Payment Methods Breakdown
	var paymentStats struct {
		PaidCash     float64 `json:"paid_cash"`
		PaidTransfer float64 `json:"paid_transfer"`
		DebtAmount   float64 `json:"debt_amount"`
	}
	database.DB.Table("sales").
		Select(`
			COALESCE(SUM(CASE WHEN payment_method = 'จ่ายเงินสด' THEN (CASE WHEN pay >= net_price THEN net_price ELSE pay END) ELSE 0 END), 0) as paid_cash,
			COALESCE(SUM(CASE WHEN payment_method = 'โอนจ่าย' THEN (CASE WHEN pay >= net_price THEN net_price ELSE pay END) ELSE 0 END), 0) as paid_transfer
		`).
		Where("shop_id = ? AND created_at >= ? AND created_at <= ?", shopID, startDate, endDate).
		Scan(&paymentStats)

	// debt_amount ต้องหักด้วยเงินที่ลูกหนี้จ่ายคืนทีหลังผ่านตาราง debt_payments ด้วย
	// (ใช้ FIFO waterfall แบบเดียวกับ Aging Report ด้านล่าง) ไม่ใช่แค่ net_price - pay ตอนขาย
	// เพราะคอลัมน์ sales.pay ไม่เคยถูกอัปเดตย้อนหลังตอนลูกหนี้มาชำระเพิ่ม
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

	// 3. Top 5 Products
	type TopProduct struct {
		ProductName string  `json:"product_name"`
		TotalQty    int     `json:"total_qty"`
		TotalSales  float64 `json:"total_sales"`
	}
	var topProducts []TopProduct
	database.DB.Table("sale_items").
		Select("products.name as product_name, SUM(sale_items.amount) as total_qty, SUM(sale_items.total_price) as total_sales").
		Joins("JOIN sales ON sales.sale_id = sale_items.sale_id").
		Joins("JOIN products ON products.product_id = sale_items.product_id").
		Where("sales.shop_id = ? AND sales.created_at >= ? AND sales.created_at <= ?", shopID, startDate, endDate).
		Group("products.product_id, products.name").
		Order("total_qty DESC").
		Limit(5).
		Scan(&topProducts)

	// 4. Debt Summary
	var debtSummary struct {
		TotalOutstanding float64 `json:"total_outstanding"`
		CollectedThisMonth float64 `json:"collected_this_month"`
		DebtorCount int64 `json:"debtor_count"`
	}
	// Total Outstanding Debt + count of debtors currently owing (across all debtors)
	database.DB.Table("debtors").
		Select("COALESCE(SUM(current_debt), 0) as total_outstanding, COUNT(CASE WHEN current_debt > 0 THEN 1 END) as debtor_count").
		Where("shop_id = ?", shopID).
		Scan(&debtSummary)

	// Collected this period (uses the selected start and end dates)
	database.DB.Table("debt_payments").
		Select("COALESCE(SUM(amount_paid), 0) as collected_this_month").
		Joins("JOIN debtors ON debtors.debtor_id = debt_payments.debtor_id").
		Where("debtors.shop_id = ? AND DATE(payment_date) >= ? AND DATE(payment_date) <= ?", shopID, startDate, endDate).
		Scan(&debtSummary.CollectedThisMonth)

	// 5. Aging Report (from sales that are unpaid)
	// Remaining balance is derived with a FIFO waterfall: debt_payments are
	// recorded per-debtor (not per-sale), so total payments are applied against a
	// debtor's oldest sales first via a running cumulative-debt window function.
	// This keeps aging amounts in sync when a debtor pays off older debts later,
	// instead of relying on the never-updated sales.pay column.
	//
	// Aging is tracked per DEBTOR (not per individual sale/bill): since a repayment
	// can't be tied to a specific bill, "age" is reset to the debtor's most recent
	// payment date whenever they pay anything (even partially). If they haven't paid
	// since their oldest still-open bill was created, age falls back to that bill's
	// date. GREATEST() picks whichever is more recent, so a brand-new bill created
	// after an old, unrelated payment still ages from its own date, not the stale
	// payment. All date comparisons are wrapped in DATE() because debt_payments.payment_date
	// stores a full timestamp — comparing it raw against a date-only asOfDate string would
	// silently exclude same-day payments made after midnight until the next calendar day.
	//
	// "Most recent payment" only counts payments made since the debtor's current debt
	// cycle started: debt_events/running_balance/cycle_start replay the debtor's full
	// history (every debt-creating sale as +delta, every payment as -delta) in
	// chronological order to find the last time their balance fully returned to zero.
	// Payments made before that "closing" point belong to an already-settled cycle and
	// must not reset the aging clock for a brand-new, unrelated debt taken on afterward.
	//
	// Aging always reflects real-time status "as of today", regardless of which
	// month/year the report page's period picker is set to — it is not a historical
	// snapshot of that period. Only the other report sections (sales chart, summary,
	// debt collection statement, etc.) are scoped to the selected start/end date.
	asOfDate := time.Now().Format("2006-01-02")
	var aging struct {
		Safe    float64 `json:"safe"`    // 1-15 days
		Warning float64 `json:"warning"` // 16-30 days
		Danger  float64 `json:"danger"`  // >30 days
	}
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
		shopID, asOfDate, shopID, asOfDate, // debt_events
		shopID, asOfDate, // debtor_paid_in_cycle
		asOfDate, asOfDate, asOfDate, // final SELECT
	).Scan(&aging)

	// 6. Top 5 Debtors
	type TopDebtor struct {
		DebtorID    int     `json:"debtor_id"`
		Name        string  `json:"name"`
		CurrentDebt float64 `json:"current_debt"`
	}
	var topDebtors []TopDebtor
	database.DB.Table("debtors").
		Select("debtor_id, name, current_debt").
		Where("shop_id = ? AND current_debt > 0", shopID).
		Order("current_debt DESC").
		Limit(5).
		Scan(&topDebtors)

	// 7. Debt Collection Statement (based on selected range)
	var debtCollection struct {
		NewDebt       float64 `json:"new_debt"`
		CollectedDebt float64 `json:"collected_debt"`
	}
	database.DB.Table("sales").
		Select("COALESCE(SUM(net_price - pay), 0) as new_debt").
		Where("shop_id = ? AND DATE(created_at) >= ? AND DATE(created_at) <= ? AND (pay < net_price OR payment_method = 'ค้างชำระ')", shopID, startDate, endDate).
		Scan(&debtCollection.NewDebt)

	database.DB.Table("debt_payments").
		Select("COALESCE(SUM(amount_paid), 0) as collected_debt").
		Joins("JOIN debtors ON debtors.debtor_id = debt_payments.debtor_id").
		Where("debtors.shop_id = ? AND DATE(payment_date) >= ? AND DATE(payment_date) <= ?", shopID, startDate, endDate).
		Scan(&debtCollection.CollectedDebt)

	// Combine into response
	c.JSON(http.StatusOK, gin.H{
		"sales_chart":     salesChart,
		"summary_stats":   summaryStats,
		"payment_methods": paymentStats,
		"top_products":    topProducts,
		"debt_summary":    debtSummary,
		"aging_report":    aging,
		"top_debtors":     topDebtors,
		"debt_collection": debtCollection,
	})
}

// GetAgingReportDetail returns the debtors behind each aging bucket
// (safe/warning/danger) shown in GetAdvancedReport's "aging_report" totals, so
// the UI can drill into "who owes what" per bucket. Reuses the exact same
// per-debtor FIFO-waterfall + debt_since logic as GetAdvancedReport's aging
// section so SUM(amount_owed) per bucket here reconciles with
// aging_report.safe/warning/danger.
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

	type AgingDebtor struct {
		DebtorID    int     `json:"debtor_id"`
		Name        string  `json:"name"`
		Phone       string  `json:"phone"`
		ImgDebtor   string  `json:"img_debtor"`
		DebtSince   string  `json:"debt_since"`
		AmountOwed  float64 `json:"amount_owed"`
		DaysOverdue int     `json:"days_overdue"`
		Bucket      string  `json:"-"`
	}

	// Aging always reflects real-time status "as of today" — see the matching
	// asOfDate comment in GetAdvancedReport.
	asOfDate := time.Now().Format("2006-01-02")

	var debtors []AgingDebtor
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

	safe := []AgingDebtor{}
	warning := []AgingDebtor{}
	danger := []AgingDebtor{}
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
