package main

import (
    "fmt"
    "log"
    "net/http"
    "os"

    "sme_hardware/backend/database"
    "sme_hardware/backend/routes"
)

func main() {
    database.ConnectDatabase()
    defer database.CloseDatabase()

    mux := routes.SetupRoutes()

    port := os.Getenv("PORT")
    if port == "" {
        port = "8081"
    }

    fmt.Printf("SME Hardware API running on :%s\n", port)

    if err := http.ListenAndServe(":"+port, mux); err != nil {
        log.Fatal(err)
    }
}
