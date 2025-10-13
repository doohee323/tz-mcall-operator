package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	serverURL := "https://jenkins.drillquiz.com/mcp-server/mcp"
	username := "admin"
	token := "11197fa40f409842983025803948aa6bcc"
	
	credentials := username + ":" + token
	authValue := "Basic " + base64.StdEncoding.EncodeToString([]byte(credentials))
	
	client := &http.Client{Timeout: 30 * time.Second}
	var sessionID string
	
	sendRequest := func(requestData map[string]interface{}) ([]byte, http.Header, error) {
		jsonData, _ := json.Marshal(requestData)
		
		req, _ := http.NewRequest("POST", serverURL, bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", authValue)
		req.Header.Set("Accept", "text/event-stream, application/json")
		
		if sessionID != "" {
			req.Header.Set("mcp-session-id", sessionID)
		}
		
		resp, err := client.Do(req)
		if err != nil {
			return nil, nil, err
		}
		defer resp.Body.Close()
		
		body, _ := io.ReadAll(resp.Body)
		return body, resp.Header, nil
	}
	
	// Initialize
	fmt.Println("🔹 Initialize...")
	initRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "tz-mcall-operator",
				"version": "1.0.0",
			},
		},
	}
	
	initBody, initHeaders, _ := sendRequest(initRequest)
	sessionID = initHeaders.Get("mcp-session-id")
	fmt.Printf("✅ Session ID: %s\n", sessionID)
	
	// Notifications
	fmt.Println("🔹 Notifications...")
	notifyRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}
	sendRequest(notifyRequest)
	
	// Call tool with LIMIT=2 (적은 데이터)
	fmt.Println("🔹 Call getJobs with limit=2...")
	toolRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "getJobs",
			"arguments": map[string]interface{}{
				"offset": 0,
				"limit":  2,  // Only 2 jobs!
			},
		},
	}
	
	toolBody, _, _ := sendRequest(toolRequest)
	
	// Parse response
	var mcpResponse struct {
		Result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	
	if err := json.Unmarshal(toolBody, &mcpResponse); err != nil {
		fmt.Printf("❌ JSON parsing failed: %v\n", err)
		fmt.Printf("Raw response length: %d bytes\n", len(toolBody))
		return
	}
	
	fmt.Printf("✅ Parsed successfully! Content items: %d\n\n", len(mcpResponse.Result.Content))
	
	// Display each job summary
	for i, content := range mcpResponse.Result.Content {
		if content.Type == "text" {
			var job map[string]interface{}
			if err := json.Unmarshal([]byte(content.Text), &job); err == nil {
				fmt.Printf("📋 Job %d: %s\n", i+1, job["name"])
				fmt.Printf("   URL: %s\n", job["url"])
				fmt.Printf("   Buildable: %v\n", job["buildable"])
				if lastBuild, ok := job["lastBuild"].(map[string]interface{}); ok {
					fmt.Printf("   Last Build: #%v\n", lastBuild["number"])
				}
				fmt.Println()
			}
		}
	}
}





