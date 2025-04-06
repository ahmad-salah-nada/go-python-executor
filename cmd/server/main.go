package main

import (
	"fmt"
	"go--python-executor/internal/handler"
	"log"
	"net/http"
)

func main() {
	// Register the execute handler
	http.HandleFunc("/execute", handler.ExecuteHandler)

	// Start the server
	port := ":8080"
	fmt.Printf("Server starting on port %s...\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
