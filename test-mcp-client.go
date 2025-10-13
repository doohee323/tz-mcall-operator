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
	// Jenkins MCP Server 설정
	serverURL := "https://jenkins.drillquiz.com/mcp-server/mcp"
	username := "admin"
	token := "11197fa40f409842983025803948aa6bcc"
	
	// Base64 인코딩
	credentials := username + ":" + token
	authValue := "Basic " + base64.StdEncoding.EncodeToString([]byte(credentials))
	
	client := &http.Client{Timeout: 30 * time.Second}
	
	var sessionID string
	
	// Helper function
	sendRequest := func(requestData map[string]interface{}) ([]byte, http.Header, error) {
		jsonData, err := json.Marshal(requestData)
		if err != nil {
			return nil, nil, err
		}
		
		fmt.Printf("\n📤 Sending request:\n%s\n", string(jsonData))
		
		req, err := http.NewRequest("POST", serverURL, bytes.NewReader(jsonData))
		if err != nil {
			return nil, nil, err
		}
		
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", authValue)
		req.Header.Set("Accept", "text/event-stream, application/json")
		
		// Add session ID if available
		if sessionID != "" {
			req.Header.Set("mcp-session-id", sessionID)
			fmt.Printf("🔑 Using session ID: %s\n", sessionID)
		}
		
		resp, err := client.Do(req)
		if err != nil {
			return nil, nil, err
		}
		defer resp.Body.Close()
		
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, err
		}
		
		fmt.Printf("📥 Response Status: %d\n", resp.StatusCode)
		
		// Print response headers
		if resp.Header.Get("mcp-session-id") != "" {
			fmt.Printf("🔑 Session ID from response: %s\n", resp.Header.Get("mcp-session-id"))
		}
		
		if resp.StatusCode >= 400 {
			return body, resp.Header, fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		
		return body, resp.Header, nil
	}
	
	// Step 1: Initialize
	fmt.Println("🔹 Step 1: Initialize handshake")
	initRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}
	
	initBody, initHeaders, err := sendRequest(initRequest)
	if err != nil {
		fmt.Printf("❌ Initialize failed: %v\n", err)
		fmt.Printf("Response body: %s\n", string(initBody))
		return
	}
	
	// Extract session ID from response header
	if sid := initHeaders.Get("mcp-session-id"); sid != "" {
		sessionID = sid
		fmt.Printf("🔑 Session ID extracted: %s\n", sessionID)
	}
	
	fmt.Printf("✅ Initialize successful\n")
	var initResponse map[string]interface{}
	json.Unmarshal(initBody, &initResponse)
	prettyInit, _ := json.MarshalIndent(initResponse, "", "  ")
	fmt.Printf("%s\n", string(prettyInit))
	
	// Step 2: Notifications/initialized
	fmt.Println("\n🔹 Step 2: Send notifications/initialized")
	notifyRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}
	
	_, _, err = sendRequest(notifyRequest)
	if err != nil {
		fmt.Printf("⚠️  Notifications failed (non-critical): %v\n", err)
	} else {
		fmt.Printf("✅ Notifications sent\n")
	}
	
	// Step 3: Call getJobs tool
	fmt.Println("\n🔹 Step 3: Call getJobs tool")
	toolRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "getJobs",
			"arguments": map[string]interface{}{
				"offset": 0,
				"limit":  10,
			},
		},
	}
	
	toolBody, _, err := sendRequest(toolRequest)
	if err != nil {
		fmt.Printf("❌ Tool call failed: %v\n", err)
		fmt.Printf("Response body: %s\n", string(toolBody))
		return
	}
	
	fmt.Printf("✅ Tool call successful\n")
	fmt.Printf("📄 Raw response:\n%s\n\n", string(toolBody))
	
	var toolResponse map[string]interface{}
	if err := json.Unmarshal(toolBody, &toolResponse); err != nil {
		fmt.Printf("❌ Failed to parse response: %v\n", err)
		return
	}
	
	prettyTool, _ := json.MarshalIndent(toolResponse, "", "  ")
	fmt.Printf("📊 Parsed response:\n%s\n", string(prettyTool))
}

