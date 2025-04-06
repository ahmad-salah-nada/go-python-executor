package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"go--python-executor/internal/models"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	// Read Python code from a file
	code, err := os.ReadFile("../../code.py")
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	// Create request payload
	payload := models.RequestPayload{Code: string(code)}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Send HTTPS POST request
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	resp, err := http.Post("https://localhost/execute", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response: %v", err)
	}

	// Parse response
	var response models.ResponsePayload
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Print response
	fmt.Println("Server Response:")
	if response.Error != "" {
		fmt.Println("Error:", response.Error)
	}
	if response.Stderr != "" {
		fmt.Println("Stderr:", response.Stderr)
	}
	if response.Stdout != "" {
		fmt.Println("Stdout:", response.Stdout)
	}
}
