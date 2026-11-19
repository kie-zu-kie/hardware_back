package models

import "time"

type StockUpdate struct {
	StockUpdateID int       `json:"stock_update_id"`
	ProductID     int       `json:"product_id"`
	ProductName   string    `json:"product_name"`
	SupplierID    int       `json:"supplier_id"`
	SupplierName string    `json:"supplier_name"`
	UpdateType   string    `json:"update_type"`
	Quantity     int       `json:"quantity"`
	UpdateDate   time.Time `json:"update_date"`
	Reason       string    `json:"reason"`
}

type StockUpdatesResponse struct {
	Success      bool         `json:"success"`
	Message      string       `json:"message"`
	StockUpdates []StockUpdate `json:"stock_updates,omitempty"`
}

type StockUpdateRequest struct {
	ProductID  int    `json:"product_id"`
	SupplierID int    `json:"supplier_id"`
	Quantity   int    `json:"quantity"`
	Reason     string `json:"reason"`
}

type StockUpdateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}