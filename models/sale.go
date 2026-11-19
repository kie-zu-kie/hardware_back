package models

type SaleItemRequest struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

type CreateSaleRequest struct {
	CustomerID    *int             `json:"customer_id"`
	CashierID     int              `json:"cashier_id"`
	PaymentMethod string           `json:"payment_method"`
	Items         []SaleItemRequest `json:"items"`
}

// Used when displaying the sales list.
type Sale struct {
	SaleID        int     `json:"sale_id"`
	CustomerID    *int    `json:"customer_id,omitempty"`
	CustomerName  string  `json:"customer_name"`
	CashierID     int     `json:"cashier_id"`
	CashierName   string  `json:"cashier_name"`
	SaleDate      string  `json:"sale_date"`
	TotalAmount   float64 `json:"total_amount"`
	PaymentMethod string  `json:"payment_method"`
}

// Used for individual sale details.
type SaleItem struct {
	SaleItemID  int     `json:"sale_item_id"`
	ProductID   int     `json:"product_id"`
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Subtotal    float64 `json:"subtotal"`
}

type SaleDetails struct {
	Sale
	Items []SaleItem `json:"items"`
}

type SalesResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Sales   []Sale `json:"sales,omitempty"`
}

type SaleDetailsResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message"`
	Sale    *SaleDetails `json:"sale,omitempty"`
}

type SaleResponse struct {
	Success     bool    `json:"success"`
	Message     string  `json:"message"`
	SaleID      int     `json:"sale_id,omitempty"`
	TotalAmount float64 `json:"total_amount,omitempty"`
}