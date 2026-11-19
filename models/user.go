package models

type Cashier struct {
    CashierID int    `json:"cashier_id"`
    FullName  string `json:"full_name"`
    Username  string `json:"username"`
    Phone     string `json:"phone"`
    Position  string `json:"position"`
    Password  string `json:"-"`
}

type LoginRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

type LoginResponse struct {
    Success bool    `json:"success"`
    Message string  `json:"message"`
    Cashier *Cashier `json:"cashier,omitempty"`
}
