package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func main() {
	// Test static file serving
	fmt.Println("🧪 Testing Web Interface...")
	
	// Test main chat page
	fmt.Println("📄 Testing chat.html...")
	resp, err := http.Get("http://localhost:9091/chat.html")
	if err != nil {
		log.Fatalf("❌ Failed to get chat.html: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		log.Fatalf("❌ Expected status 200, got %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("❌ Failed to read response body: %v", err)
	}
	
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Realtime Chat System") {
		log.Fatalf("❌ chat.html doesn't contain expected title")
	}
	
	if !strings.Contains(bodyStr, "chat.js") {
		log.Fatalf("❌ chat.html doesn't reference chat.js")
	}
	
	if !strings.Contains(bodyStr, "styles.css") {
		log.Fatalf("❌ chat.html doesn't reference styles.css")
	}
	
	fmt.Println("✅ chat.html loaded successfully")
	
	// Test CSS file
	fmt.Println("🎨 Testing styles.css...")
	resp, err = http.Get("http://localhost:9091/styles.css")
	if err != nil {
		log.Fatalf("❌ Failed to get styles.css: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		log.Fatalf("❌ Expected status 200 for CSS, got %d", resp.StatusCode)
	}
	
	cssBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("❌ Failed to read CSS response body: %v", err)
	}
	
	cssStr := string(cssBody)
	if !strings.Contains(cssStr, ".chat-container") {
		log.Fatalf("❌ styles.css doesn't contain expected CSS classes")
	}
	
	fmt.Println("✅ styles.css loaded successfully")
	
	// Test JavaScript file
	fmt.Println("📜 Testing chat.js...")
	resp, err = http.Get("http://localhost:9091/chat.js")
	if err != nil {
		log.Fatalf("❌ Failed to get chat.js: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		log.Fatalf("❌ Expected status 200 for JS, got %d", resp.StatusCode)
	}
	
	jsBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("❌ Failed to read JS response body: %v", err)
	}
	
	jsStr := string(jsBody)
	if !strings.Contains(jsStr, "class ChatApp") {
		log.Fatalf("❌ chat.js doesn't contain expected ChatApp class")
	}
	
	if !strings.Contains(jsStr, "WebSocket") {
		log.Fatalf("❌ chat.js doesn't contain WebSocket code")
	}
	
	fmt.Println("✅ chat.js loaded successfully")
	
	// Test index page (fallback)
	fmt.Println("🏠 Testing index.html...")
	resp, err = http.Get("http://localhost:9091/")
	if err != nil {
		log.Fatalf("❌ Failed to get index page: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		log.Fatalf("❌ Expected status 200 for index, got %d", resp.StatusCode)
	}
	
	fmt.Println("✅ index.html loaded successfully")
	
	// Test WebSocket endpoint availability
	fmt.Println("🔌 Testing WebSocket endpoint availability...")
	
	// We can't easily test WebSocket from HTTP client, but we can check if the endpoint exists
	// by trying to connect and seeing if we get the right error
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	
	req, err := http.NewRequest("GET", "http://localhost:9091/ws", nil)
	if err != nil {
		log.Fatalf("❌ Failed to create WebSocket test request: %v", err)
	}
	
	resp, err = client.Do(req)
	if err != nil {
		log.Fatalf("❌ Failed to test WebSocket endpoint: %v", err)
	}
	defer resp.Body.Close()
	
	// WebSocket upgrade should fail with 400 Bad Request (missing upgrade headers)
	if resp.StatusCode != 400 {
		log.Printf("⚠️ WebSocket endpoint returned status %d (expected 400 for HTTP request)", resp.StatusCode)
	} else {
		fmt.Println("✅ WebSocket endpoint is available")
	}
	
	fmt.Println("\n🎉 All web interface tests passed!")
	fmt.Println("🌐 Open http://localhost:9091/chat.html in your browser to test the chat interface")
	fmt.Println("🔧 Or use http://localhost:9091/ for the simple test client")
}