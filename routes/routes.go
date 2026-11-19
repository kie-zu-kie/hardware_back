package routes

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"sme_hardware/backend/handlers"
)

func SetupRoutes() http.Handler {
	mux := http.NewServeMux()

	// --------------------------------------------------
	// LOGIN
	// --------------------------------------------------

	mux.HandleFunc(
		"/api/login",
		withCORS(handlers.Login),
	)

	// --------------------------------------------------
	// CUSTOMERS
	// --------------------------------------------------

	mux.HandleFunc(
		"/api/customers",
		withCORS(func(w http.ResponseWriter, r *http.Request) {

			if r.Method == http.MethodGet {
				handlers.GetCustomers(w, r)
				return
			}

			if r.Method == http.MethodPost {
				handlers.AddCustomer(w, r)
				return
			}

			w.WriteHeader(http.StatusMethodNotAllowed)
		}),
	)

	mux.HandleFunc(
		"/api/customers/",
		withCORS(func(w http.ResponseWriter, r *http.Request) {

			if r.Method == http.MethodDelete &&
				strings.TrimPrefix(
					r.URL.Path,
					"/api/customers/",
				) != "" {

				handlers.DeleteCustomer(w, r)
				return
			}

			w.WriteHeader(http.StatusMethodNotAllowed)
		}),
	)

	// --------------------------------------------------
	// SUPPLIERS
	// --------------------------------------------------

	mux.HandleFunc(
		"/api/suppliers",
		withCORS(func(w http.ResponseWriter, r *http.Request) {

			if r.Method == http.MethodGet {
				handlers.GetSuppliers(w, r)
				return
			}

			if r.Method == http.MethodPost {
				handlers.AddSupplier(w, r)
				return
			}

			w.WriteHeader(http.StatusMethodNotAllowed)
		}),
	)

	mux.HandleFunc(
		"/api/suppliers/",
		withCORS(func(w http.ResponseWriter, r *http.Request) {

			if r.Method == http.MethodDelete {
				handlers.DeleteSupplier(w, r)
				return
			}

			w.WriteHeader(http.StatusMethodNotAllowed)
		}),
	)

	// --------------------------------------------------
	// SALES
	// --------------------------------------------------

	mux.HandleFunc("/api/sales", withCORS(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetSales(w, r)

		case http.MethodPost:
			handlers.CreateSale(w, r)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/sales/", withCORS(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.GetSaleDetails(w, r)
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	// --------------------------------------------------
	// PRODUCTS
	// --------------------------------------------------

	mux.HandleFunc(
		"/api/products",
		withCORS(func(w http.ResponseWriter, r *http.Request) {

			switch r.Method {

			case http.MethodGet:
				handlers.GetProducts(w, r)

			case http.MethodPost:
				handlers.AddProduct(w, r)

			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		}),
	)

	mux.HandleFunc(
		"/api/products/",
		withCORS(func(w http.ResponseWriter, r *http.Request) {

			switch r.Method {

			case http.MethodPut:
				handlers.UpdateProduct(w, r)

			case http.MethodDelete:
				handlers.DeleteProduct(w, r)

			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		}),
	)

	// --------------------------------------------------
	// STOCK UPDATES
	// --------------------------------------------------

	mux.HandleFunc(
		"/api/stock-updates",
		withCORS(func(w http.ResponseWriter, r *http.Request) {

			if r.Method == http.MethodGet {
				handlers.GetStockUpdates(w, r)
				return
			}

			if r.Method == http.MethodPost {
				handlers.AddStock(w, r)
				return
			}

			w.WriteHeader(http.StatusMethodNotAllowed)
		}),
	)

	// --------------------------------------------------
	// ROOT
	// --------------------------------------------------

	mux.HandleFunc(
		"/",
		func(w http.ResponseWriter, r *http.Request) {

			w.Header().Set(
				"Content-Type",
				"text/plain",
			)

			w.WriteHeader(http.StatusOK)

			_, _ = w.Write(
				[]byte("SME Hardware API is running"),
			)
		},
	)

	return mux
}

// --------------------------------------------------
// CORS
// --------------------------------------------------

func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		origin := r.Header.Get("Origin")

		if isAllowedOrigin(origin) {
			w.Header().Set(
				"Access-Control-Allow-Origin",
				origin,
			)

			w.Header().Add(
				"Vary",
				"Origin",
			)
		}

		// Allowed HTTP methods
		w.Header().Set(
			"Access-Control-Allow-Methods",
			"GET, POST, PUT, DELETE, OPTIONS",
		)

		// Allowed request headers
		w.Header().Set(
			"Access-Control-Allow-Headers",
			"Content-Type, X-Cashier-ID",
		)

		// Handle browser preflight request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

// --------------------------------------------------
// ALLOWED ORIGIN CHECK (LOCAL & DEPLOYED ORIGINS)
// --------------------------------------------------

func isAllowedOrigin(origin string) bool {
	if origin == "" {
		return false
	}

	parsed, err := url.Parse(origin)
	if err != nil || parsed.Hostname() == "" {
		return false
	}

	host := strings.ToLower(parsed.Hostname())

	// Local development
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}

	// Any Vercel deployment preview or production domain
	if host == "vercel.app" || strings.HasSuffix(host, ".vercel.app") {
		return true
	}

	// Environment variable overrides (e.g. FRONTEND_URL or ALLOWED_ORIGINS)
	if frontendURL := os.Getenv("FRONTEND_URL"); frontendURL != "" {
		if strings.TrimRight(strings.ToLower(frontendURL), "/") == strings.TrimRight(strings.ToLower(origin), "/") {
			return true
		}
	}

	if allowedOrigins := os.Getenv("ALLOWED_ORIGINS"); allowedOrigins != "" {
		for _, allowed := range strings.Split(allowedOrigins, ",") {
			if strings.TrimSpace(strings.ToLower(allowed)) == strings.TrimRight(strings.ToLower(origin), "/") {
				return true
			}
		}
	}

	return false
}
