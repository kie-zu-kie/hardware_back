package models

type Product struct {
	ProductID    int     `json:"product_id"`
	ProductName  string  `json:"product_name"`
	Category     string  `json:"category"`
	Unit         string  `json:"unit"`
	SellingPrice float64 `json:"selling_price"`
	CurrentStock int     `json:"current_stock"`
	ReorderLevel int     `json:"reorder_level"`
	SupplierID   int     `json:"supplier_id"`
	SupplierName string  `json:"supplier_name,omitempty"`
}

type ProductRequest struct {
	ProductName  string  `json:"product_name"`
	Category     string  `json:"category"`
	Unit         string  `json:"unit"`
	SellingPrice float64 `json:"selling_price"`
	CurrentStock int     `json:"current_stock"`
	ReorderLevel int     `json:"reorder_level"`
	SupplierID   int     `json:"supplier_id"`
}

type ProductsResponse struct {
	Success  bool      `json:"success"`
	Message  string    `json:"message"`
	Products []Product `json:"products,omitempty"`
}

type ProductResponse struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Product *Product `json:"product,omitempty"`
}