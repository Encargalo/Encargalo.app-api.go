package dtos

type StatsPeriod string

const (
	PeriodDay   StatsPeriod = "day"
	PeriodWeek  StatsPeriod = "week"
	PeriodMonth StatsPeriod = "month"
)

type PeriodMetrics struct {
	TotalEarned         float64 `json:"total_earned"`
	TotalLostRejections float64 `json:"total_lost_rejections"`
	OrdersCompleted     int     `json:"orders_completed"`
	OrdersRejected      int     `json:"orders_rejected"`
	RejectionRate       float64 `json:"rejection_rate"`
}

type StatsSummaryResponse struct {
	Period         StatsPeriod   `json:"period"`
	Current        PeriodMetrics `json:"current"`
	Previous       PeriodMetrics `json:"previous"`
	CurrentBalance float64       `json:"current_balance"`
}

type TopProductItem struct {
	ItemID       string  `json:"item_id"`
	Name         string  `json:"name"`
	Image        string  `json:"image"`
	QuantitySold int     `json:"quantity_sold"`
	TotalRevenue float64 `json:"total_revenue"`
}

type TopProductsResponse struct {
	Period   StatsPeriod      `json:"period"`
	Current  []TopProductItem `json:"current"`
	Previous []TopProductItem `json:"previous"`
}

type SalesDataPoint struct {
	Date            string  `json:"date"`
	Earned          float64 `json:"earned"`
	Lost            float64 `json:"lost"`
	OrdersCompleted int     `json:"orders_completed"`
	OrdersRejected  int     `json:"orders_rejected"`
}

type SalesOverTimeResponse struct {
	Period      StatsPeriod      `json:"period"`
	Granularity string           `json:"granularity"`
	NextCursor  *string          `json:"next_cursor"`
	Current     []SalesDataPoint `json:"current"`
	Previous    []SalesDataPoint `json:"previous"`
}

type CustomerMetrics struct {
	NewCustomers       int `json:"new_customers"`
	ReturningCustomers int `json:"returning_customers"`
}

type CustomerStatsResponse struct {
	Period   StatsPeriod     `json:"period"`
	Current  CustomerMetrics `json:"current"`
	Previous CustomerMetrics `json:"previous"`
}
