package handlers

import (
    "context"
    "encoding/json"
    "net/http"
    "strings"

    "sme_hardware/backend/database"
    "sme_hardware/backend/models"
)

func Login(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        writeJSON(w, http.StatusMethodNotAllowed, models.LoginResponse{
            Success: false,
            Message: "Method not allowed",
        })
        return
    }

    var request models.LoginRequest

    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        writeJSON(w, http.StatusBadRequest, models.LoginResponse{
            Success: false,
            Message: "Invalid JSON request",
        })
        return
    }

    request.Username = strings.TrimSpace(request.Username)

    if request.Username == "" || request.Password == "" {
        writeJSON(w, http.StatusBadRequest, models.LoginResponse{
            Success: false,
            Message: "Username and password are required",
        })
        return
    }

    var cashier models.Cashier

    query := `
        SELECT cashier_id, full_name, username, phone, position, password
        FROM cashiers
        WHERE username = $1
        LIMIT 1
    `

    err := database.DB.QueryRow(
        context.Background(),
        query,
        request.Username,
    ).Scan(
        &cashier.CashierID,
        &cashier.FullName,
        &cashier.Username,
        &cashier.Phone,
        &cashier.Position,
        &cashier.Password,
    )

    if err != nil {
        writeJSON(w, http.StatusUnauthorized, models.LoginResponse{
            Success: false,
            Message: "Invalid username or password",
        })
        return
    }

    // This matches your current database if passwords are stored as plain text.
    // We should change this to bcrypt once the basic login is working.
    if request.Password != cashier.Password {
        writeJSON(w, http.StatusUnauthorized, models.LoginResponse{
            Success: false,
            Message: "Invalid username or password",
        })
        return
    }

    cashier.Password = ""

    writeJSON(w, http.StatusOK, models.LoginResponse{
        Success: true,
        Message: "Login successful",
        Cashier: &cashier,
    })
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(data)
}
