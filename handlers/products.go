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

// GET /api/products
func GetProducts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, models.ProductsResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	query := `
		SELECT
			p.product_id,
			p.product_name,
			p.category,
			p.unit,
			p.selling_price,
			p.current_stock,
			p.reorder_level,
			p.supplier_id,
			COALESCE(s.supplier_name, '') AS supplier_name
		FROM products p
		LEFT JOIN suppliers s
			ON p.supplier_id = s.supplier_id
		ORDER BY p.product_id ASC
	`

	rows, err := database.DB.Query(context.Background(), query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ProductsResponse{
			Success: false,
			Message: "Failed to retrieve products: " + err.Error(),
		})
		return
	}

	defer rows.Close()

	products := []models.Product{}

	for rows.Next() {
		var product models.Product

		err := rows.Scan(
			&product.ProductID,
			&product.ProductName,
			&product.Category,
			&product.Unit,
			&product.SellingPrice,
			&product.CurrentStock,
			&product.ReorderLevel,
			&product.SupplierID,
			&product.SupplierName,
		)

		if err != nil {
			writeJSON(w, http.StatusInternalServerError, models.ProductsResponse{
				Success: false,
				Message: "Failed to read product data: " + err.Error(),
			})
			return
		}

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ProductsResponse{
			Success: false,
			Message: "Failed to read product data: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, models.ProductsResponse{
		Success:  true,
		Message:  "Products retrieved successfully",
		Products: products,
	})
}

// Check whether a cashier exists and is a Store Manager.
func isManager(cashierID int) bool {
	var position string

	err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT position
		FROM cashiers
		WHERE cashier_id = $1
		LIMIT 1
		`,
		cashierID,
	).Scan(&position)

	if err != nil {
		return false
	}

	return strings.EqualFold(strings.TrimSpace(position), "Store Manager")
}

// POST /api/products
func AddProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, models.ProductResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	cashierID, err := strconv.Atoi(r.Header.Get("X-Cashier-ID"))

	if err != nil || cashierID <= 0 {
		writeJSON(w, http.StatusUnauthorized, models.ProductResponse{
			Success: false,
			Message: "Manager authentication required",
		})
		return
	}

	if !isManager(cashierID) {
		writeJSON(w, http.StatusForbidden, models.ProductResponse{
			Success: false,
			Message: "Only managers can add products",
		})
		return
	}

	var request models.ProductRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ProductResponse{
			Success: false,
			Message: "Invalid JSON request",
		})
		return
	}

	request.ProductName = strings.TrimSpace(request.ProductName)
	request.Category = strings.TrimSpace(request.Category)
	request.Unit = strings.TrimSpace(request.Unit)

	if request.ProductName == "" ||
		request.Category == "" ||
		request.Unit == "" ||
		request.SellingPrice < 0 ||
		request.CurrentStock < 0 ||
		request.ReorderLevel < 0 ||
		request.SupplierID <= 0 {

		writeJSON(w, http.StatusBadRequest, models.ProductResponse{
			Success: false,
			Message: "Invalid product information",
		})
		return
	}

	ctx := context.Background()

	var supplierExists bool

	err = database.DB.QueryRow(
		ctx,
		`
		SELECT EXISTS(
			SELECT 1
			FROM suppliers
			WHERE supplier_id = $1
		)
		`,
		request.SupplierID,
	).Scan(&supplierExists)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ProductResponse{
			Success: false,
			Message: "Failed to check supplier: " + err.Error(),
		})
		return
	}

	if !supplierExists {
		writeJSON(w, http.StatusBadRequest, models.ProductResponse{
			Success: false,
			Message: "Supplier does not exist",
		})
		return
	}

	var product models.Product

	err = database.DB.QueryRow(
		ctx,
		`
		INSERT INTO products (
			product_name,
			category,
			unit,
			selling_price,
			current_stock,
			reorder_level,
			supplier_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING
			product_id,
			product_name,
			category,
			unit,
			selling_price,
			current_stock,
			reorder_level,
			supplier_id
		`,
		request.ProductName,
		request.Category,
		request.Unit,
		request.SellingPrice,
		request.CurrentStock,
		request.ReorderLevel,
		request.SupplierID,
	).Scan(
		&product.ProductID,
		&product.ProductName,
		&product.Category,
		&product.Unit,
		&product.SellingPrice,
		&product.CurrentStock,
		&product.ReorderLevel,
		&product.SupplierID,
	)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ProductResponse{
			Success: false,
			Message: "Failed to add product: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusCreated, models.ProductResponse{
		Success: true,
		Message: "Product added successfully",
		Product: &product,
	})
}

// PUT /api/products/{id}
func UpdateProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeJSON(w, http.StatusMethodNotAllowed, models.ProductResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	cashierID, err := strconv.Atoi(r.Header.Get("X-Cashier-ID"))

	if err != nil || cashierID <= 0 {
		writeJSON(w, http.StatusUnauthorized, models.ProductResponse{
			Success: false,
			Message: "Manager authentication required",
		})
		return
	}

	if !isManager(cashierID) {
		writeJSON(w, http.StatusForbidden, models.ProductResponse{
			Success: false,
			Message: "Only managers can update products",
		})
		return
	}

	idText := strings.TrimPrefix(r.URL.Path, "/api/products/")

	productID, err := strconv.Atoi(idText)

	if err != nil || productID <= 0 {
		writeJSON(w, http.StatusBadRequest, models.ProductResponse{
			Success: false,
			Message: "Invalid product ID",
		})
		return
	}

	var request models.ProductRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ProductResponse{
			Success: false,
			Message: "Invalid JSON request",
		})
		return
	}

	request.ProductName = strings.TrimSpace(request.ProductName)
	request.Category = strings.TrimSpace(request.Category)
	request.Unit = strings.TrimSpace(request.Unit)

	if request.ProductName == "" ||
		request.Category == "" ||
		request.Unit == "" ||
		request.SellingPrice < 0 ||
		request.CurrentStock < 0 ||
		request.ReorderLevel < 0 ||
		request.SupplierID <= 0 {

		writeJSON(w, http.StatusBadRequest, models.ProductResponse{
			Success: false,
			Message: "Invalid product information",
		})
		return
	}

	ctx := context.Background()

	var supplierExists bool

	err = database.DB.QueryRow(
		ctx,
		`
		SELECT EXISTS(
			SELECT 1
			FROM suppliers
			WHERE supplier_id = $1
		)
		`,
		request.SupplierID,
	).Scan(&supplierExists)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ProductResponse{
			Success: false,
			Message: "Failed to check supplier: " + err.Error(),
		})
		return
	}

	if !supplierExists {
		writeJSON(w, http.StatusBadRequest, models.ProductResponse{
			Success: false,
			Message: "Supplier does not exist",
		})
		return
	}

	var product models.Product

	err = database.DB.QueryRow(
		ctx,
		`
		UPDATE products
		SET
			product_name = $1,
			category = $2,
			unit = $3,
			selling_price = $4,
			current_stock = $5,
			reorder_level = $6,
			supplier_id = $7
		WHERE product_id = $8
		RETURNING
			product_id,
			product_name,
			category,
			unit,
			selling_price,
			current_stock,
			reorder_level,
			supplier_id
		`,
		request.ProductName,
		request.Category,
		request.Unit,
		request.SellingPrice,
		request.CurrentStock,
		request.ReorderLevel,
		request.SupplierID,
		productID,
	).Scan(
		&product.ProductID,
		&product.ProductName,
		&product.Category,
		&product.Unit,
		&product.SellingPrice,
		&product.CurrentStock,
		&product.ReorderLevel,
		&product.SupplierID,
	)

	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			writeJSON(w, http.StatusNotFound, models.ProductResponse{
				Success: false,
				Message: "Product not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, models.ProductResponse{
			Success: false,
			Message: "Failed to update product: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, models.ProductResponse{
		Success: true,
		Message: "Product updated successfully",
		Product: &product,
	})
}

// DELETE /api/products/{id}
func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, models.ProductResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	cashierID, err := strconv.Atoi(r.Header.Get("X-Cashier-ID"))

	if err != nil || cashierID <= 0 {
		writeJSON(w, http.StatusUnauthorized, models.ProductResponse{
			Success: false,
			Message: "Manager authentication required",
		})
		return
	}

	if !isManager(cashierID) {
		writeJSON(w, http.StatusForbidden, models.ProductResponse{
			Success: false,
			Message: "Only managers can delete products",
		})
		return
	}

	idText := strings.TrimPrefix(r.URL.Path, "/api/products/")

	productID, err := strconv.Atoi(idText)

	if err != nil || productID <= 0 {
		writeJSON(w, http.StatusBadRequest, models.ProductResponse{
			Success: false,
			Message: "Invalid product ID",
		})
		return
	}

	result, err := database.DB.Exec(
		context.Background(),
		`
		DELETE FROM products
		WHERE product_id = $1
		`,
		productID,
	)

	if err != nil {
		writeJSON(w, http.StatusConflict, models.ProductResponse{
			Success: false,
			Message: "Product cannot be deleted. It may be used by existing sales or stock records.",
		})
		return
	}

	if result.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, models.ProductResponse{
			Success: false,
			Message: "Product not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, models.ProductResponse{
		Success: true,
		Message: "Product deleted successfully",
	})
}