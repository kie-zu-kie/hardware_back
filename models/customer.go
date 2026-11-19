package models

type Customer struct {
	CustomerID   int    `json:"customer_id"`
	CustomerName string `json:"customer_name"`
	Phone        string `json:"phone"`
	Address      string `json:"address"`
	CustomerType string `json:"customer_type"`
}

type CreateCustomerRequest struct {
	CustomerName string `json:"customer_name"`
	Phone        string `json:"phone"`
	Address      string `json:"address"`
	CustomerType string `json:"customer_type"`
}

type CustomersResponse struct {
	Success   bool       `json:"success"`
	Message   string     `json:"message"`
	Customers []Customer `json:"customers,omitempty"`
}

type CustomerResponse struct {
	Success  bool      `json:"success"`
	Message  string    `json:"message"`
	Customer *Customer `json:"customer,omitempty"`
}