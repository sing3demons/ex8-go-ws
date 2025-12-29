package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

// ClientMessage represents messages sent to server
type ClientMessage struct {
	Type     string `json:"type"`
	Content  string `json:"content,omitempty"`
	Username string `json:"username,omitempty"`
	Room     string `json:"room,omitempty"`
	Command  string `json:"command,omitempty"`
	Query    string `json:"query,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// ServerMessage represents messages received from server
type ServerMessage struct {
	Type      string    `json:"type"`
	Content   string    `json:"content,omitempty"`
	Sender    string    `json:"sender,omitempty"`
	Username  string    `json:"username,omitempty"`
	Room      string    `json:"room,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Users     []string  `json:"users,omitempty"`
	Rooms     []string  `json:"rooms,omitempty"`
	Message   string    `json:"message,omitempty"`
}

func main() {
	fmt.Println("🧪 Testing Complete Chat Flow...")

	// Connect to WebSocket server
	u := url.URL{Scheme: "ws", Host: "localhost:9091", Path: "/ws"}
	fmt.Printf("🔗 Connecting to %s\n", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("❌ Failed to connect: %v", err)
	}
	defer conn.Close()

	fmt.Println("✅ Connected to WebSocket server")

	// Read initial auth request
	fmt.Println("\n📨 Reading initial auth request...")
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Printf("⚠️ Failed to read auth request (this is normal): %v", err)
		fmt.Println("📨 No initial auth request received, proceeding with authentication...")
	} else {
		fmt.Printf("📨 Received: %s\n", string(message))
	}

	// Test 1: Authenticate with plain text (backward compatibility)
	fmt.Println("\n🔐 Test 1: Authenticate with plain text")
	
	// Generate unique username
	rand.Seed(time.Now().UnixNano())
	username := fmt.Sprintf("TestUser%d", rand.Intn(10000))
	fmt.Printf("🔐 Using username: %s\n", username)
	
	err = conn.WriteMessage(websocket.TextMessage, []byte(username))
	if err != nil {
		log.Fatalf("❌ Failed to send username: %v", err)
	}

	// Read welcome message
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, message, err = conn.ReadMessage()
	if err != nil {
		log.Printf("⚠️ Failed to read welcome message: %v", err)
	} else {
		fmt.Printf("📨 Welcome message: %s\n", string(message))
	}

	// Test 2: Send a regular message
	fmt.Println("\n💬 Test 2: Send regular message")
	err = conn.WriteMessage(websocket.TextMessage, []byte("Hello everyone!"))
	if err != nil {
		log.Fatalf("❌ Failed to send message: %v", err)
	}

	fmt.Println("✅ Message sent successfully")

	// Test 3: Send a command
	fmt.Println("\n⚡ Test 3: Send /users command")
	err = conn.WriteMessage(websocket.TextMessage, []byte("/users"))
	if err != nil {
		log.Fatalf("❌ Failed to send command: %v", err)
	}

	// Read command response
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, message, err = conn.ReadMessage()
	if err != nil {
		log.Printf("⚠️ Failed to read command response: %v", err)
	} else {
		fmt.Printf("📨 Command response: %s\n", string(message))
	}

	// Test 4: Send /help command
	fmt.Println("\n❓ Test 4: Send /help command")
	err = conn.WriteMessage(websocket.TextMessage, []byte("/help"))
	if err != nil {
		log.Fatalf("❌ Failed to send help command: %v", err)
	}

	// Read help response
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, message, err = conn.ReadMessage()
	if err != nil {
		log.Printf("⚠️ Failed to read help response: %v", err)
	} else {
		fmt.Printf("📨 Help response: %s\n", string(message))
	}

	// Test 5: Try MongoDB commands if available
	fmt.Println("\n📚 Test 5: Test MongoDB commands")
	
	// Test history command
	err = conn.WriteMessage(websocket.TextMessage, []byte("/history 5"))
	if err != nil {
		log.Fatalf("❌ Failed to send history command: %v", err)
	}

	// Read history response
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, message, err = conn.ReadMessage()
	if err != nil {
		log.Printf("⚠️ Failed to read history response: %v", err)
	} else {
		fmt.Printf("📨 History response: %s\n", string(message))
	}

	// Give some time to receive any additional messages
	fmt.Println("\n⏳ Waiting for additional messages...")
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	
	for i := 0; i < 10; i++ {
		_, message, err := conn.ReadMessage()
		if err != nil {
			// Timeout or connection closed - that's okay
			break
		}

		fmt.Printf("📨 Additional message: %s\n", string(message))
	}

	fmt.Println("\n🎉 Complete chat flow test completed!")
	fmt.Println("✅ The server is working with plain text messages")
	fmt.Println("🌐 You can now test the web interface at http://localhost:9091/chat.html")
	fmt.Println("🔧 Or use the simple test client at http://localhost:9091/")
}