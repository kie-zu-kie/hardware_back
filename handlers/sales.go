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

// POST /api/sales
func CreateSale(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, models.SaleResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	var request models.CreateSaleRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, models.SaleResponse{
			Success: false,
			Message: "Invalid JSON request",
		})
		return
	}

	if request.CashierID <= 0 {
		writeJSON(w, http.StatusBadRequest, models.SaleResponse{
			Success: false,
			Message: "Invalid cashier",
		})
		return
	}

	if len(request.Items) == 0 {
		writeJSON(w, http.StatusBadRequest, models.SaleResponse{
			Success: false,
			Message: "Sale must contain at least one product",
		})
		return
	}

	request.PaymentMethod = strings.TrimSpace(request.PaymentMethod)

	if request.PaymentMethod != "Cash" &&
		request.PaymentMethod != "GCash" &&
		request.PaymentMethod != "Card" {

		writeJSON(w, http.StatusBadRequest, models.SaleResponse{
			Success: false,
			Message: "Invalid payment method",
		})
		return
	}

	ctx := context.Background()

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.SaleResponse{
			Success: false,
			Message: "Failed to start transaction: " + err.Error(),
		})
		return
	}

	defer tx.Rollback(ctx)

	// =========================================================
	// CHECK CASHIER
	// =========================================================

	var cashierExists bool

	err = tx.QueryRow(
		ctx,
		`
		SELECT EXISTS(
			SELECT 1
			FROM cashiers
			WHERE cashier_id = $1
		)
		`,
		request.CashierID,
	).Scan(&cashierExists)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.SaleResponse{
			Success: false,
			Message: "Failed to check cashier: " + err.Error(),
		})
		return
	}

	if !cashierExists {
		writeJSON(w, http.StatusBadRequest, models.SaleResponse{
			Success: false,
			Message: "Cashier does not exist",
		})
		return
	}

	// =========================================================
	// HANDLE CUSTOMER
	// =========================================================

	// Keep walk-in sales unassigned. A nil customer ID is intentionally stored
	// as NULL so it cannot resolve to a real customer in the sales history.
	var customerID *int

	if request.CustomerID != nil {
		if *request.CustomerID <= 0 {
			writeJSON(w, http.StatusBadRequest, models.SaleResponse{
				Success: false,
				Message: "Invalid customer",
			})
			return
		}

		var customerExists bool
		err = tx.QueryRow(
			ctx,
			`
			SELECT EXISTS(
				SELECT 1
				FROM customers
				WHERE customer_id = $1
			)
			`,
			*request.CustomerID,
		).Scan(&customerExists)

		if err != nil {
			writeJSON(w, http.StatusInternalServerError, models.SaleResponse{
				Success: false,
				Message: "Failed to check customer: " + err.Error(),
			})
			return
		}

		if !customerExists {
			writeJSON(w, http.StatusBadRequest, models.SaleResponse{
				Success: false,
				Message: "Customer does not exist",
			})
			return
		}

		customerID = request.CustomerID
	}

	// =========================================================
	// CREATE SALE
	// =========================================================

	var saleID int

	err = tx.QueryRow(
		ctx,
		`
		INSERT INTO sales (
			customer_id,
			cashier_id,
			sale_date,
			total_amount,
			payment_method
		)
		VALUES (
			$1,
			$2,
			CURRENT_TIMESTAMP,
			0,
			$3
		)
		RETURNING sale_id
		`,
		customerID,
		request.CashierID,
		request.PaymentMethod,
	).Scan(&saleID)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.SaleResponse{
			Success: false,
			Message: "Failed to create sale: " + err.Error(),
		})
		return
	}

	// =========================================================
	// PROCESS SALE ITEMS
	// =========================================================

	var totalAmount float64

	for _, item := range request.Items {

		if item.ProductID <= 0 || item.Quantity <= 0 {
			writeJSON(w, http.StatusBadRequest, models.SaleResponse{
				Success: false,
				Message: "Invalid product or quantity",
			})
			return
		}

		var (
			productName  string
			unitPrice    float64
			currentStock int
		)

		err = tx.QueryRow(
			ctx,
			`
			SELECT
				product_name,
				selling_price,
				current_stock
			FROM products
			WHERE product_id = $1
			FOR UPDATE
			`,
			item.ProductID,
		).Scan(
			&productName,
			&unitPrice,
			&currentStock,
		)

		if err != nil {
			writeJSON(w, http.StatusBadRequest, models.SaleResponse{
				Success: false,
				Message: "Product not found: " + strconv.Itoa(item.ProductID),
			})
			return
		}

		// =====================================================
		// CHECK STOCK
		// =====================================================

		if currentStock < item.Quantity {
			writeJSON(w, http.StatusBadRequest, models.SaleResponse{
				Success: false,
				Message: "Insufficient stock for " + productName,
			})
			return
		}

		// =====================================================
		// CALCULATE SUBTOTAL
		// =====================================================

		subtotal := unitPrice * float64(item.Quantity)

		totalAmount += subtotal

		// =====================================================
		// SAVE SALE ITEM
		// =====================================================

		_, err = tx.Exec(
			ctx,
			`
			INSERT INTO sale_items (
				sale_id,
				product_id,
				quantity,
				unit_price,
				subtotal
			)
			VALUES ($1, $2, $3, $4, $5)
			`,
			saleID,
			item.ProductID,
			item.Quantity,
			unitPrice,
			subtotal,
		)

		if err != nil {
			writeJSON(w, http.StatusInternalServerError, models.SaleResponse{
				Success: false,
				Message: "Failed to save sale item: " + err.Error(),
			})
			return
		}

		// =====================================================
		// UPDATE PRODUCT STOCK
		// =====================================================

		_, err = tx.Exec(
			ctx,
			`
			UPDATE products
			SET current_stock = current_stock - $1
			WHERE product_id = $2
			`,
			item.Quantity,
			item.ProductID,
		)

		if err != nil {
			writeJSON(w, http.StatusInternalServerError, models.SaleResponse{
				Success: false,
				Message: "Failed to update product stock: " + err.Error(),
			})
			return
		}
	}

	// =========================================================
	// UPDATE SALE TOTAL
	// =========================================================

	_, err = tx.Exec(
		ctx,
		`
		UPDATE sales
		SET total_amount = $1
		WHERE sale_id = $2
		`,
		totalAmount,
		saleID,
	)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.SaleResponse{
			Success: false,
			Message: "Failed to update sale total: " + err.Error(),
		})
		return
	}

	// =========================================================
	// COMMIT
	// =========================================================

	if err := tx.Commit(ctx); err != nil {
		writeJSON(w, http.StatusInternalServerError, models.SaleResponse{
			Success: false,
			Message: "Failed to save sale: " + err.Error(),
		})
		return
	}

	// =========================================================
	// SUCCESS
	// =========================================================

	writeJSON(w, http.StatusCreated, models.SaleResponse{
		Success:     true,
		Message:     "Sale created successfully",
		SaleID:      saleID,
		TotalAmount: totalAmount,
	})
}

// =============================================================
// GET /api/sales
// =============================================================

func GetSales(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, models.SalesResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	query := `
		SELECT
			s.sale_id,
			s.customer_id,
			COALESCE(c.customer_name, 'Walk-in Customer') AS customer_name,
			s.cashier_id,
			COALESCE(ca.full_name, '') AS cashier_name,
			TO_CHAR(s.sale_date, 'YYYY-MM-DD HH24:MI:SS') AS sale_date,
			s.total_amount,
			s.payment_method
		FROM sales s
		LEFT JOIN customers c
			ON s.customer_id = c.customer_id
		LEFT JOIN cashiers ca
			ON s.cashier_id = ca.cashier_id
		ORDER BY s.sale_date DESC, s.sale_id DESC
	`

	rows, err := database.DB.Query(
		context.Background(),
		query,
	)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.SalesResponse{
			Success: false,
			Message: "Failed to retrieve sales: " + err.Error(),
		})
		return
	}

	defer rows.Close()

	sales := []models.Sale{}

	for rows.Next() {

		var sale models.Sale

		err := rows.Scan(
			&sale.SaleID,
			&sale.CustomerID,
			&sale.CustomerName,
			&sale.CashierID,
			&sale.CashierName,
			&sale.SaleDate,
			&sale.TotalAmount,
			&sale.PaymentMethod,
		)

		if err != nil {
			writeJSON(w, http.StatusInternalServerError, models.SalesResponse{
				Success: false,
				Message: "Failed to read sale data: " + err.Error(),
			})
			return
		}

		sales = append(sales, sale)
	}

	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, models.SalesResponse{
			Success: false,
			Message: "Failed to read sale data: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, models.SalesResponse{
		Success: true,
		Message: "Sales retrieved successfully",
		Sales:   sales,
	})
}

// =============================================================
// GET /api/sales/{id}
// =============================================================

func GetSaleDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, models.SaleDetailsResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	idText := strings.TrimPrefix(
		r.URL.Path,
		"/api/sales/",
	)

	saleID, err := strconv.Atoi(idText)

	if err != nil || saleID <= 0 {
		writeJSON(w, http.StatusBadRequest, models.SaleDetailsResponse{
			Success: false,
			Message: "Invalid sale ID",
		})
		return
	}

	ctx := context.Background()

	var sale models.SaleDetails

	err = database.DB.QueryRow(
		ctx,
		`
		SELECT
			s.sale_id,
			s.customer_id,
			COALESCE(c.customer_name, 'Walk-in Customer'),
			s.cashier_id,
			COALESCE(ca.full_name, ''),
			TO_CHAR(s.sale_date, 'YYYY-MM-DD HH24:MI:SS'),
			s.total_amount,
			s.payment_method
		FROM sales s
		LEFT JOIN customers c
			ON s.customer_id = c.customer_id
		LEFT JOIN cashiers ca
			ON s.cashier_id = ca.cashier_id
		WHERE s.sale_id = $1
		`,
		saleID,
	).Scan(
		&sale.SaleID,
		&sale.CustomerID,
		&sale.CustomerName,
		&sale.CashierID,
		&sale.CashierName,
		&sale.SaleDate,
		&sale.TotalAmount,
		&sale.PaymentMethod,
	)

	if err != nil {
		writeJSON(w, http.StatusNotFound, models.SaleDetailsResponse{
			Success: false,
			Message: "Sale not found",
		})
		return
	}

	rows, err := database.DB.Query(
		ctx,
		`
		SELECT
			si.sale_item_id,
			si.product_id,
			p.product_name,
			si.quantity,
			si.unit_price,
			si.subtotal
		FROM sale_items si
		INNER JOIN products p
			ON si.product_id = p.product_id
		WHERE si.sale_id = $1
		ORDER BY si.sale_item_id ASC
		`,
		saleID,
	)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.SaleDetailsResponse{
			Success: false,
			Message: "Failed to retrieve sale items: " + err.Error(),
		})
		return
	}

	defer rows.Close()

	sale.Items = []models.SaleItem{}

	for rows.Next() {

		var item models.SaleItem

		err := rows.Scan(
			&item.SaleItemID,
			&item.ProductID,
			&item.ProductName,
			&item.Quantity,
			&item.UnitPrice,
			&item.Subtotal,
		)

		if err != nil {
			writeJSON(w, http.StatusInternalServerError, models.SaleDetailsResponse{
				Success: false,
				Message: "Failed to read sale items: " + err.Error(),
			})
			return
		}

		sale.Items = append(
			sale.Items,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, models.SaleDetailsResponse{
			Success: false,
			Message: "Failed to read sale items: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, models.SaleDetailsResponse{
		Success: true,
		Message: "Sale details retrieved successfully",
		Sale:    &sale,
	})
}
