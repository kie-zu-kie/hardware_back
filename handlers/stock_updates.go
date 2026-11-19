package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"sme_hardware/backend/database"
	"sme_hardware/backend/models"
)

func GetStockUpdates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed,
			models.StockUpdatesResponse{
				Success: false,
				Message: "Method not allowed",
			},
		)
		return
	}

	ctx := context.Background()

	query := `
		SELECT
			s.stock_update_id,
			s.product_id,
			p.product_name,
			s.supplier_id,
			su.supplier_name,
			s.update_type,
			s.quantity,
			s.update_date,
			s.reason
		FROM stock_updates s
		LEFT JOIN products p
			ON s.product_id = p.product_id
		LEFT JOIN suppliers su
			ON s.supplier_id = su.supplier_id
		ORDER BY s.update_date DESC
	`

	rows, err := database.DB.Query(ctx, query)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError,
			models.StockUpdatesResponse{
				Success: false,
				Message: "Failed to retrieve stock updates: " + err.Error(),
			},
		)
		return
	}

	defer rows.Close()

	stockUpdates := []models.StockUpdate{}

	for rows.Next() {
		var item models.StockUpdate

		err := rows.Scan(
			&item.StockUpdateID,
			&item.ProductID,
			&item.ProductName,
			&item.SupplierID,
			&item.SupplierName,
			&item.UpdateType,
			&item.Quantity,
			&item.UpdateDate,
			&item.Reason,
		)

		if err != nil {
			writeJSON(w, http.StatusInternalServerError,
				models.StockUpdatesResponse{
					Success: false,
					Message: "Failed to read stock update row: " + err.Error(),
				},
			)
			return
		}

		stockUpdates = append(stockUpdates, item)
	}

	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError,
			models.StockUpdatesResponse{
				Success: false,
				Message: "Failed to read stock updates: " + err.Error(),
			},
		)
		return
	}

	writeJSON(w, http.StatusOK,
		models.StockUpdatesResponse{
			Success:      true,
			Message:      "Stock updates retrieved successfully",
			StockUpdates: stockUpdates,
		},
	)
}

func AddStock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed,
			models.StockUpdateResponse{
				Success: false,
				Message: "Method not allowed",
			},
		)
		return
	}

	var request models.StockUpdateRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest,
			models.StockUpdateResponse{
				Success: false,
				Message: "Invalid request",
			},
		)
		return
	}

	request.Reason = strings.TrimSpace(request.Reason)

	if request.ProductID <= 0 ||
		request.SupplierID <= 0 ||
		request.Quantity <= 0 {

		writeJSON(w, http.StatusBadRequest,
			models.StockUpdateResponse{
				Success: false,
				Message: "Invalid product, supplier, or quantity",
			},
		)
		return
	}

	ctx := context.Background()

	tx, err := database.DB.Begin(ctx)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError,
			models.StockUpdateResponse{
				Success: false,
				Message: "Failed to start transaction",
			},
		)
		return
	}

	defer tx.Rollback(ctx)

	var productExists bool

	err = tx.QueryRow(
		ctx,
		`SELECT EXISTS(
			SELECT 1
			FROM products
			WHERE product_id = $1
		)`,
		request.ProductID,
	).Scan(&productExists)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError,
			models.StockUpdateResponse{
				Success: false,
				Message: "Failed to check product",
			},
		)
		return
	}

	if !productExists {
		writeJSON(w, http.StatusBadRequest,
			models.StockUpdateResponse{
				Success: false,
				Message: "Product does not exist",
			},
		)
		return
	}

	var supplierExists bool

	err = tx.QueryRow(
		ctx,
		`SELECT EXISTS(
			SELECT 1
			FROM suppliers
			WHERE supplier_id = $1
		)`,
		request.SupplierID,
	).Scan(&supplierExists)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError,
			models.StockUpdateResponse{
				Success: false,
				Message: "Failed to check supplier",
			},
		)
		return
	}

	if !supplierExists {
		writeJSON(w, http.StatusBadRequest,
			models.StockUpdateResponse{
				Success: false,
				Message: "Supplier does not exist",
			},
		)
		return
	}

	_, err = tx.Exec(
		ctx,
		`
		UPDATE products
		SET current_stock = current_stock + $1
		WHERE product_id = $2
		`,
		request.Quantity,
		request.ProductID,
	)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError,
			models.StockUpdateResponse{
				Success: false,
				Message: "Failed to update product stock",
			},
		)
		return
	}

	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO stock_updates
		(
			product_id,
			supplier_id,
			update_type,
			quantity,
			update_date,
			reason
		)
		VALUES
		(
			$1,
			$2,
			'IN',
			$3,
			CURRENT_TIMESTAMP,
			$4
		)
		`,
		request.ProductID,
		request.SupplierID,
		request.Quantity,
		request.Reason,
	)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError,
			models.StockUpdateResponse{
				Success: false,
				Message: "Failed to save stock update",
			},
		)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		writeJSON(w, http.StatusInternalServerError,
			models.StockUpdateResponse{
				Success: false,
				Message: "Failed to complete stock update",
			},
		)
		return
	}

	writeJSON(w, http.StatusOK,
		models.StockUpdateResponse{
			Success: true,
			Message: "Stock updated successfully",
		},
	)
}