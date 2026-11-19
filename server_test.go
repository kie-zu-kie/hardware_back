package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"sme_hardware/backend/database"
	"sme_hardware/backend/models"
	"sme_hardware/backend/routes"
)

var testRouter http.Handler

func TestMain(m *testing.M) {
	database.ConnectDatabase()
	testRouter = routes.SetupRoutes()
	code := m.Run()
	database.CloseDatabase()
	os.Exit(code)
}

// 1. Root / Health Check Test
func TestHealthCheck(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	testRouter.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	expected := "SME Hardware API is running"
	if rr.Body.String() != expected {
		t.Fatalf("expected body %q, got %q", expected, rr.Body.String())
	}
}

// 2. CORS Middleware Test
func TestCORS(t *testing.T) {
	t.Run("Allowed local origin preflight", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/products", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		rr := httptest.NewRecorder()

		testRouter.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Errorf("expected status 204 for OPTIONS, got %d", rr.Code)
		}
		if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
			t.Errorf("expected Access-Control-Allow-Origin 'http://localhost:5173', got %q", got)
		}
	})

	t.Run("Allowed Vercel origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/products", nil)
		req.Header.Set("Origin", "https://sme-hardware.vercel.app")
		rr := httptest.NewRecorder()

		testRouter.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Errorf("expected status 204 for OPTIONS, got %d", rr.Code)
		}
		if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://sme-hardware.vercel.app" {
			t.Errorf("expected Access-Control-Allow-Origin 'https://sme-hardware.vercel.app', got %q", got)
		}
	})

	t.Run("Disallowed foreign origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/products", nil)
		req.Header.Set("Origin", "http://malicious-domain.com")
		rr := httptest.NewRecorder()

		testRouter.ServeHTTP(rr, req)

		if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("expected empty Access-Control-Allow-Origin for untrusted domain, got %q", got)
		}
	})
}

// 3. Login Endpoint Tests
func TestLogin(t *testing.T) {
	t.Run("Method Not Allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/login", nil)
		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected status 405, got %d", rr.Code)
		}
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewBufferString("not-json"))
		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("Empty Username or Password", func(t *testing.T) {
		body := `{"username": "", "password": ""}`
		req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("Invalid Credentials", func(t *testing.T) {
		body := `{"username": "john.reyes", "password": "wrongpassword"}`
		req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rr.Code)
		}
	})

	t.Run("Successful Store Manager Login", func(t *testing.T) {
		body := `{"username": "john.reyes", "password": "101"}`
		req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
		}

		var resp models.LoginResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !resp.Success {
			t.Fatalf("expected success true, got false")
		}
		if resp.Cashier == nil {
			t.Fatalf("expected cashier in response, got nil")
		}
		if resp.Cashier.Position != "Store Manager" {
			t.Errorf("expected position 'Store Manager', got %q", resp.Cashier.Position)
		}
	})

	t.Run("Successful Cashier Login", func(t *testing.T) {
		body := `{"username": "mark.santos", "password": "102"}`
		req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
		}

		var resp models.LoginResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !resp.Success || resp.Cashier == nil || resp.Cashier.Position != "Cashier" {
			t.Fatalf("expected valid Cashier response, got %+v", resp)
		}
	})
}

// 4. Customers Endpoint Tests
func TestCustomers(t *testing.T) {
	t.Run("Get Customers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/customers", nil)
		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		var resp models.CustomersResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to parse json: %v", err)
		}
		if !resp.Success {
			t.Fatalf("expected success true")
		}
		if len(resp.Customers) == 0 {
			t.Logf("warning: customers table is currently empty")
		}
	})

	t.Run("Customer Lifecycle (Create, Delete, Verify)", func(t *testing.T) {
		// Create customer
		newCust := models.CreateCustomerRequest{
			CustomerName: "Test Automation Customer",
			Phone:        "09990001122",
			Address:      "123 Test Street",
			CustomerType: "Regular",
		}
		bodyBytes, _ := json.Marshal(newCust)
		createReq := httptest.NewRequest(http.MethodPost, "/api/customers", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, createReq)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected status 201 Created, got %d: %s", rr.Code, rr.Body.String())
		}

		var createResp models.CustomerResponse
		if err := json.NewDecoder(rr.Body).Decode(&createResp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if createResp.Customer == nil || createResp.Customer.CustomerID <= 0 {
			t.Fatalf("expected valid customer id, got %+v", createResp)
		}

		createdID := createResp.Customer.CustomerID

		// Delete customer
		delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/customers/%d", createdID), nil)
		rrDel := httptest.NewRecorder()
		testRouter.ServeHTTP(rrDel, delReq)

		if rrDel.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK for delete, got %d: %s", rrDel.Code, rrDel.Body.String())
		}

		// Delete again -> should return 404
		delAgainReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/customers/%d", createdID), nil)
		rrDelAgain := httptest.NewRecorder()
		testRouter.ServeHTTP(rrDelAgain, delAgainReq)

		if rrDelAgain.Code != http.StatusNotFound {
			t.Fatalf("expected status 404 for deleted customer, got %d", rrDelAgain.Code)
		}
	})
}

// 5. Suppliers Endpoint Tests
func TestSuppliers(t *testing.T) {
	t.Run("Get Suppliers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/suppliers", nil)
		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		var resp models.SuppliersResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to parse json: %v", err)
		}
		if !resp.Success {
			t.Fatalf("expected success true")
		}
	})

	t.Run("Supplier Lifecycle (Create, Delete)", func(t *testing.T) {
		newSup := models.CreateSupplierRequest{
			SupplierName:  "Test Automation Supplier",
			ContactPerson: "Tester Guy",
			Phone:         "09112223344",
			Address:       "456 Supply Road",
			Email:         "tester@supplier.com",
		}
		bodyBytes, _ := json.Marshal(newSup)
		createReq := httptest.NewRequest(http.MethodPost, "/api/suppliers", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, createReq)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected status 201 Created, got %d: %s", rr.Code, rr.Body.String())
		}

		var createResp models.SupplierResponse
		if err := json.NewDecoder(rr.Body).Decode(&createResp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if createResp.Supplier == nil || createResp.Supplier.SupplierID <= 0 {
			t.Fatalf("expected valid supplier id, got %+v", createResp)
		}

		createdID := createResp.Supplier.SupplierID

		// Delete supplier
		delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/suppliers/%d", createdID), nil)
		rrDel := httptest.NewRecorder()
		testRouter.ServeHTTP(rrDel, delReq)

		if rrDel.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK for delete, got %d: %s", rrDel.Code, rrDel.Body.String())
		}
	})
}

// 6. Products Endpoint Tests
func TestProducts(t *testing.T) {
	t.Run("Get Products", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/products", nil)
		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		var resp models.ProductsResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to parse json: %v", err)
		}
		if !resp.Success {
			t.Fatalf("expected success true")
		}
	})

	t.Run("Add Product Auth Checks", func(t *testing.T) {
		prodReq := models.ProductRequest{
			ProductName:  "Test Unauthorized Product",
			Category:     "Tools",
			Unit:         "pcs",
			SellingPrice: 50.0,
			CurrentStock: 10,
			ReorderLevel: 5,
			SupplierID:   1,
		}
		bodyBytes, _ := json.Marshal(prodReq)

		// 1. Missing X-Cashier-ID
		req1 := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewBuffer(bodyBytes))
		rr1 := httptest.NewRecorder()
		testRouter.ServeHTTP(rr1, req1)
		if rr1.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", rr1.Code)
		}

		// 2. Non-Manager X-Cashier-ID (Mark Santos ID: 2 is Cashier)
		req2 := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewBuffer(bodyBytes))
		req2.Header.Set("X-Cashier-ID", "2")
		rr2 := httptest.NewRecorder()
		testRouter.ServeHTTP(rr2, req2)
		if rr2.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for Cashier, got %d", rr2.Code)
		}
	})

	t.Run("Product Lifecycle (Create, Update, Delete)", func(t *testing.T) {
		prodReq := models.ProductRequest{
			ProductName:  "Automated Test Drill Bit",
			Category:     "Power Tools",
			Unit:         "set",
			SellingPrice: 299.50,
			CurrentStock: 25,
			ReorderLevel: 5,
			SupplierID:   1,
		}
		bodyBytes, _ := json.Marshal(prodReq)

		// Create with Manager (John Reyes ID: 1)
		createReq := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewBuffer(bodyBytes))
		createReq.Header.Set("X-Cashier-ID", "1")
		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, createReq)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected status 201 Created, got %d: %s", rr.Code, rr.Body.String())
		}

		var createResp models.ProductResponse
		if err := json.NewDecoder(rr.Body).Decode(&createResp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if createResp.Product == nil || createResp.Product.ProductID <= 0 {
			t.Fatalf("expected valid product id, got %+v", createResp)
		}

		createdID := createResp.Product.ProductID

		// Update Product
		prodReq.SellingPrice = 349.00
		prodReq.CurrentStock = 30
		updateBytes, _ := json.Marshal(prodReq)

		updateReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/products/%d", createdID), bytes.NewBuffer(updateBytes))
		updateReq.Header.Set("X-Cashier-ID", "1")
		rrUpdate := httptest.NewRecorder()
		testRouter.ServeHTTP(rrUpdate, updateReq)

		if rrUpdate.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK for update, got %d: %s", rrUpdate.Code, rrUpdate.Body.String())
		}

		// Delete Product
		delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/products/%d", createdID), nil)
		delReq.Header.Set("X-Cashier-ID", "1")
		rrDel := httptest.NewRecorder()
		testRouter.ServeHTTP(rrDel, delReq)

		if rrDel.Code != http.StatusOK {
			t.Fatalf("expected status 200 OK for delete, got %d: %s", rrDel.Code, rrDel.Body.String())
		}
	})
}

// 7. Stock Updates Endpoint Tests
func TestStockUpdates(t *testing.T) {
	t.Run("Get Stock Updates", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/stock-updates", nil)
		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		var resp models.StockUpdatesResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to parse json: %v", err)
		}
		if !resp.Success {
			t.Fatalf("expected success true")
		}
	})

	t.Run("Add Stock Validation", func(t *testing.T) {
		// Non-existent product ID
		body := `{"product_id": 999999, "supplier_id": 1, "quantity": 10, "reason": "Restock"}`
		req := httptest.NewRequest(http.MethodPost, "/api/stock-updates", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for non-existent product, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

// 8. Sales Endpoint Tests
func TestSales(t *testing.T) {
	t.Run("Get Sales", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/sales", nil)
		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		var resp models.SalesResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to parse json: %v", err)
		}
		if !resp.Success {
			t.Fatalf("expected success true")
		}
	})

	t.Run("Create Sale Validation", func(t *testing.T) {
		// Empty items
		body := `{"cashier_id": 1, "payment_method": "Cash", "items": []}`
		req := httptest.NewRequest(http.MethodPost, "/api/sales", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for empty items, got %d", rr.Code)
		}

		// Invalid payment method
		body2 := `{"cashier_id": 1, "payment_method": "Crypto", "items": [{"product_id": 1, "quantity": 1}]}`
		req2 := httptest.NewRequest(http.MethodPost, "/api/sales", bytes.NewBufferString(body2))
		rr2 := httptest.NewRecorder()
		testRouter.ServeHTTP(rr2, req2)

		if rr2.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid payment method, got %d", rr2.Code)
		}
	})

	t.Run("Create Sale and View Details (Walk-in Customer)", func(t *testing.T) {
		// First find an existing product with stock > 0
		var prodID int
		var initialStock int
		err := database.DB.QueryRow(
			context.Background(),
			"SELECT product_id, current_stock FROM products WHERE current_stock > 5 LIMIT 1",
		).Scan(&prodID, &initialStock)

		if err != nil {
			t.Skipf("No product with stock > 5 available to test sale: %v", err)
			return
		}

		saleReq := models.CreateSaleRequest{
			CustomerID:    nil, // Walk-in
			CashierID:     1,
			PaymentMethod: "Cash",
			Items: []models.SaleItemRequest{
				{
					ProductID: prodID,
					Quantity:  1,
				},
			},
		}

		bodyBytes, _ := json.Marshal(saleReq)
		req := httptest.NewRequest(http.MethodPost, "/api/sales", bytes.NewBuffer(bodyBytes))
		rr := httptest.NewRecorder()
		testRouter.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected status 201 Created, got %d: %s", rr.Code, rr.Body.String())
		}

		var saleResp models.SaleResponse
		if err := json.NewDecoder(rr.Body).Decode(&saleResp); err != nil {
			t.Fatalf("failed to parse sale response: %v", err)
		}
		if saleResp.SaleID <= 0 {
			t.Fatalf("expected valid sale id, got %+v", saleResp)
		}

		saleID := saleResp.SaleID

		// Verify sale details endpoint GET /api/sales/{id}
		getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/sales/%d", saleID), nil)
		rrGet := httptest.NewRecorder()
		testRouter.ServeHTTP(rrGet, getReq)

		if rrGet.Code != http.StatusOK {
			t.Fatalf("expected status 200 for sale details, got %d: %s", rrGet.Code, rrGet.Body.String())
		}

		var detailsResp models.SaleDetailsResponse
		if err := json.NewDecoder(rrGet.Body).Decode(&detailsResp); err != nil {
			t.Fatalf("failed to parse details response: %v", err)
		}
		if detailsResp.Sale == nil || len(detailsResp.Sale.Items) != 1 {
			t.Fatalf("expected 1 item in sale details, got %+v", detailsResp.Sale)
		}

		// Verify stock was decremented
		var afterStock int
		_ = database.DB.QueryRow(context.Background(), "SELECT current_stock FROM products WHERE product_id = $1", prodID).Scan(&afterStock)
		if afterStock != initialStock-1 {
			t.Errorf("expected stock %d, got %d", initialStock-1, afterStock)
		}

		// Restore stock and clean up test sale
		_, _ = database.DB.Exec(context.Background(), "UPDATE products SET current_stock = $1 WHERE product_id = $2", initialStock, prodID)
		_, _ = database.DB.Exec(context.Background(), "DELETE FROM sale_items WHERE sale_id = $1", saleID)
		_, _ = database.DB.Exec(context.Background(), "DELETE FROM sales WHERE sale_id = $1", saleID)
	})
}
