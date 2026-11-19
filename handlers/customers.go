package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"sme_hardware/backend/database"
	"sme_hardware/backend/models"
)

func GetCustomers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, models.CustomersResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	query := `
		SELECT
			customer_id,
			COALESCE(customer_name, ''),
			COALESCE(phone, ''),
			COALESCE(address, ''),
			COALESCE(customer_type, '')
		FROM customers
		ORDER BY customer_id ASC
	`

	rows, err := database.DB.Query(
		context.Background(),
		query,
	)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.CustomersResponse{
			Success: false,
			Message: "Failed to retrieve customers",
		})
		return
	}

	defer rows.Close()

	customers := []models.Customer{}

	for rows.Next() {
		var customer models.Customer

		err := rows.Scan(
			&customer.CustomerID,
			&customer.CustomerName,
			&customer.Phone,
			&customer.Address,
			&customer.CustomerType,
		)

		if err != nil {
			writeJSON(w, http.StatusInternalServerError, models.CustomersResponse{
				Success: false,
				Message: "Failed to read customer data",
			})
			return
		}

		customers = append(customers, customer)
	}

	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, models.CustomersResponse{
			Success: false,
			Message: "Failed to process customer data",
		})
		return
	}

	writeJSON(w, http.StatusOK, models.CustomersResponse{
		Success:   true,
		Message:   "Customers retrieved successfully",
		Customers: customers,
	})
}

// ADD CUSTOMER
func AddCustomer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, models.CustomerResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	var request models.CreateCustomerRequest

	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, models.CustomerResponse{
			Success: false,
			Message: "Invalid JSON request",
		})
		return
	}

	request.CustomerName = strings.TrimSpace(request.CustomerName)
	request.Phone = strings.TrimSpace(request.Phone)
	request.Address = strings.TrimSpace(request.Address)
	request.CustomerType = strings.TrimSpace(request.CustomerType)

	if request.CustomerName == "" {
		writeJSON(w, http.StatusBadRequest, models.CustomerResponse{
			Success: false,
			Message: "Customer name is required",
		})
		return
	}

	if request.Phone == "" {
		writeJSON(w, http.StatusBadRequest, models.CustomerResponse{
			Success: false,
			Message: "Phone number is required",
		})
		return
	}

	if request.Address == "" {
		writeJSON(w, http.StatusBadRequest, models.CustomerResponse{
			Success: false,
			Message: "Address is required",
		})
		return
	}

	if request.CustomerType == "" {
		writeJSON(w, http.StatusBadRequest, models.CustomerResponse{
			Success: false,
			Message: "Customer type is required",
		})
		return
	}

	var customer models.Customer

	query := `
		INSERT INTO customers
			(customer_name, phone, address, customer_type)
		VALUES
			($1, $2, $3, $4)
		RETURNING
			customer_id,
			customer_name,
			phone,
			address,
			customer_type
	`

	err := database.DB.QueryRow(
		context.Background(),
		query,
		request.CustomerName,
		request.Phone,
		request.Address,
		request.CustomerType,
	).Scan(
		&customer.CustomerID,
		&customer.CustomerName,
		&customer.Phone,
		&customer.Address,
		&customer.CustomerType,
	)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.CustomerResponse{
			Success: false,
			Message: "Failed to add customer",
		})
		return
	}

	writeJSON(w, http.StatusCreated, models.CustomerResponse{
		Success:  true,
		Message:  "Customer added successfully",
		Customer: &customer,
	})
}

// DELETE CUSTOMER
func DeleteCustomer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, models.CustomerResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	// Get customer ID from the URL
	idText := strings.TrimPrefix(r.URL.Path, "/api/customers/")

	customerID, err := strconv.Atoi(idText)

	if err != nil {
		writeJSON(w, http.StatusBadRequest, models.CustomerResponse{
			Success: false,
			Message: "Invalid customer ID",
		})
		return
	}

	// Delete the customer
	query := `
		DELETE FROM customers
		WHERE customer_id = $1
	`

	result, err := database.DB.Exec(
		context.Background(),
		query,
		customerID,
	)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.CustomerResponse{
			Success: false,
			Message: "Failed to delete customer",
		})
		return
	}

	// Check if a customer was actually deleted
	if result.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, models.CustomerResponse{
			Success: false,
			Message: "Customer not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, models.CustomerResponse{
		Success: true,
		Message: "Customer deleted successfully",
	})
}

// Helper for decoding JSON
func decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}
