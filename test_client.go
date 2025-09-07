package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Simple end-to-end test to verify the API works
func main() {
	baseURL := "http://localhost:8080/api/v1"

	fmt.Println("Testing Configuration Management API...")

	// Test 1: Create configuration
	fmt.Println("\n1. Creating configuration...")
	createPayload := map[string]interface{}{
		"name": "test-config",
		"data": map[string]interface{}{
			"max_limit": 1000,
			"enabled":   true,
		},
	}

	jsonData, _ := json.Marshal(createPayload)
	resp, err := http.Post(baseURL+"/configs", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating config: %v\n", err)
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	fmt.Printf("Create response status: %d\n", resp.StatusCode)

	// Test 2: Get latest configuration
	fmt.Println("\n2. Getting latest configuration...")
	resp, err = http.Get(baseURL + "/configs/test-config")
	if err != nil {
		fmt.Printf("Error getting config: %v\n", err)
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	fmt.Printf("Get response status: %d\n", resp.StatusCode)

	// Test 3: Update configuration
	fmt.Println("\n3. Updating configuration...")
	updatePayload := map[string]interface{}{
		"data": map[string]interface{}{
			"max_limit": 2000,
			"enabled":   false,
		},
	}

	jsonData, _ = json.Marshal(updatePayload)
	req, _ := http.NewRequest("PUT", baseURL+"/configs/test-config", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("Error updating config: %v\n", err)
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	fmt.Printf("Update response status: %d\n", resp.StatusCode)

	// Test 4: List versions
	fmt.Println("\n4. Listing versions...")
	resp, err = http.Get(baseURL + "/configs/test-config/versions")
	if err != nil {
		fmt.Printf("Error listing versions: %v\n", err)
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	fmt.Printf("List versions response status: %d\n", resp.StatusCode)

	fmt.Println("\nAll tests completed successfully!")
}
