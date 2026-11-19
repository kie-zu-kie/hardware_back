package models

type Supplier struct {
	SupplierID    int    `json:"supplier_id"`
	SupplierName  string `json:"supplier_name"`
	ContactPerson string `json:"contact_person"`
	Phone         string `json:"phone"`
	Address       string `json:"address"`
	Email         string `json:"email"`
}

type CreateSupplierRequest struct {
	SupplierName  string `json:"supplier_name"`
	ContactPerson string `json:"contact_person"`
	Phone         string `json:"phone"`
	Address       string `json:"address"`
	Email         string `json:"email"`
}

type SuppliersResponse struct {
	Success   bool       `json:"success"`
	Message   string     `json:"message"`
	Suppliers []Supplier `json:"suppliers,omitempty"`
}

type SupplierResponse struct {
	Success  bool      `json:"success"`
	Message  string    `json:"message"`
	Supplier *Supplier `json:"supplier,omitempty"`
}