package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Example client demonstrating how to interact with the gateway API
func main() {
	// Health check
	resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		fmt.Printf("Error making health check request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	var healthResponse map[string]interface{}
	if err := json.Unmarshal(body, &healthResponse); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}

	fmt.Printf("Health Check Response: %+v\n", healthResponse)

	// Auth status check
	resp, err = http.Get("http://localhost:8080/auth/status")
	if err != nil {
		fmt.Printf("Error making auth status request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading auth response: %v\n", err)
		return
	}

	var authResponse map[string]interface{}
	if err := json.Unmarshal(body, &authResponse); err != nil {
		fmt.Printf("Error parsing auth JSON: %v\n", err)
		return
	}

	fmt.Printf("Auth Status Response: %+v\n", authResponse)
}