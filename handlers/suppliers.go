package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"sme_hardware/backend/database"
	"sme_hardware/backend/models"
)

func GetSuppliers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, models.SuppliersResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	query := `
		SELECT
			supplier_id,
			supplier_name,
			contact_person,
			phone,
			address,
			email
		FROM suppliers
		ORDER BY supplier_id ASC
	`

	rows, err := database.DB.Query(
		context.Background(),
		query,
	)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.SuppliersResponse{
			Success: false,
			Message: "Failed to retrieve suppliers",
		})
		return
	}

	defer rows.Close()

	suppliers := []models.Supplier{}

	for rows.Next() {
		var supplier models.Supplier

		err := rows.Scan(
			&supplier.SupplierID,
			&supplier.SupplierName,
			&supplier.ContactPerson,
			&supplier.Phone,
			&supplier.Address,
			&supplier.Email,
		)

		if err != nil {
			writeJSON(w, http.StatusInternalServerError, models.SuppliersResponse{
				Success: false,
				Message: "Failed to read supplier data",
			})
			return
		}

		suppliers = append(suppliers, supplier)
	}

	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, models.SuppliersResponse{
			Success: false,
			Message: "Failed to process supplier data",
		})
		return
	}

	writeJSON(w, http.StatusOK, models.SuppliersResponse{
		Success:   true,
		Message:   "Suppliers retrieved successfully",
		Suppliers: suppliers,
	})
}


// ADD SUPPLIER
func AddSupplier(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, models.SupplierResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	var request models.CreateSupplierRequest

	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, models.SupplierResponse{
			Success: false,
			Message: "Invalid JSON request",
		})
		return
	}

	request.SupplierName = strings.TrimSpace(request.SupplierName)
	request.ContactPerson = strings.TrimSpace(request.ContactPerson)
	request.Phone = strings.TrimSpace(request.Phone)
	request.Address = strings.TrimSpace(request.Address)
	request.Email = strings.TrimSpace(request.Email)

	if request.SupplierName == "" {
		writeJSON(w, http.StatusBadRequest, models.SupplierResponse{
			Success: false,
			Message: "Supplier name is required",
		})
		return
	}

	if request.ContactPerson == "" {
		writeJSON(w, http.StatusBadRequest, models.SupplierResponse{
			Success: false,
			Message: "Contact person is required",
		})
		return
	}

	if request.Phone == "" {
		writeJSON(w, http.StatusBadRequest, models.SupplierResponse{
			Success: false,
			Message: "Phone number is required",
		})
		return
	}

	if request.Address == "" {
		writeJSON(w, http.StatusBadRequest, models.SupplierResponse{
			Success: false,
			Message: "Address is required",
		})
		return
	}

	if request.Email == "" {
		writeJSON(w, http.StatusBadRequest, models.SupplierResponse{
			Success: false,
			Message: "Email is required",
		})
		return
	}

	var supplier models.Supplier

	query := `
		INSERT INTO suppliers
			(supplier_name, contact_person, phone, address, email)
		VALUES
			($1, $2, $3, $4, $5)
		RETURNING
			supplier_id,
			supplier_name,
			contact_person,
			phone,
			address,
			email
	`

	err := database.DB.QueryRow(
		context.Background(),
		query,
		request.SupplierName,
		request.ContactPerson,
		request.Phone,
		request.Address,
		request.Email,
	).Scan(
		&supplier.SupplierID,
		&supplier.SupplierName,
		&supplier.ContactPerson,
		&supplier.Phone,
		&supplier.Address,
		&supplier.Email,
	)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.SupplierResponse{
			Success: false,
			Message: "Failed to add supplier",
		})
		return
	}

	writeJSON(w, http.StatusCreated, models.SupplierResponse{
		Success:  true,
		Message:  "Supplier added successfully",
		Supplier: &supplier,
	})
}

// DELETE SUPPLIER
func DeleteSupplier(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, models.SupplierResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	idText := strings.TrimPrefix(
		r.URL.Path,
		"/api/suppliers/",
	)

	supplierID, err := strconv.Atoi(idText)

	if err != nil {
		writeJSON(w, http.StatusBadRequest, models.SupplierResponse{
			Success: false,
			Message: "Invalid supplier ID",
		})
		return
	}

	query := `
		DELETE FROM suppliers
		WHERE supplier_id = $1
	`

	result, err := database.DB.Exec(
		context.Background(),
		query,
		supplierID,
	)

	if err != nil {
		writeJSON(w, http.StatusConflict, models.SupplierResponse{
			Success: false,
			Message: "This supplier cannot be deleted because it is being used by other records.",
		})
		return
	}

	if result.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, models.SupplierResponse{
			Success: false,
			Message: "Supplier not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, models.SupplierResponse{
		Success: true,
		Message: "Supplier deleted successfully",
	})
}