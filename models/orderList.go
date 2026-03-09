package models

type OrderItemRequest struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Unit     string `json:"unit"`
	Note     string `json:"note"`
}

type ExportRequest struct {
	ShopID int                `json:"shop_id"` // รับแค่ ID
	Items  []OrderItemRequest `json:"items"`
}
