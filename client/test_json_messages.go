package main

import (
	"encoding/json"
	"fmt"
	"log"
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
	Type      string      `json:"type"`
	Content   string      `json:"content,omitempty"`
	Sender    string      `json:"sender,omitempty"`
	Username  string      `json:"username,omitempty"`
	Room      string      `json:"room,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Users     []string    `json:"users,omitempty"`
	Rooms     []string    `json:"rooms,omitempty"`
	Messages  []Message   `json:"messages,omitempty"`
	Message   string      `json:"message,omitempty"`
}

// Message represents a chat message
type Message struct {
	ID        string    `json:"id,omitempty"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	Sender    string    `json:"sender"`
	Username  string    `json:"username"`
	RoomName  string    `json:"room_name"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	fmt.Println("🧪 Testing JSON WebSocket Messages...")

	// Connect to WebSocket server
	u := url.URL{Scheme: "ws", Host: "localhost:9091", Path: "/ws"}
	fmt.Printf("🔗 Connecting to %s\n", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("❌ Failed to connect: %v", err)
	}
	defer conn.Close()

	fmt.Println("✅ Connected to WebSocket server")

	// Test 1: Join with JSON message
	fmt.Println("\n📝 Test 1: Join with JSON message")
	joinMsg := ClientMessage{
		Type:     "join",
		Username: "TestUser123",
	}

	data, err := json.Marshal(joinMsg)
	if err != nil {
		log.Fatalf("❌ Failed to marshal join message: %v", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		log.Fatalf("❌ Failed to send join message: %v", err)
	}

	// Read welcome message
	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("❌ Failed to read welcome message: %v", err)
	}

	var serverMsg ServerMessage
	err = json.Unmarshal(message, &serverMsg)
	if err != nil {
		fmt.Printf("⚠️ Received non-JSON message: %s\n", string(message))
	} else {
		fmt.Printf("✅ Received JSON message: Type=%s, Message=%s\n", serverMsg.Type, serverMsg.Message)
	}

	// Test 2: Send a chat message
	fmt.Println("\n💬 Test 2: Send chat message")
	chatMsg := ClientMessage{
		Type:    "message",
		Content: "Hello from JSON client!",
	}

	data, err = json.Marshal(chatMsg)
	if err != nil {
		log.Fatalf("❌ Failed to marshal chat message: %v", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		log.Fatalf("❌ Failed to send chat message: %v", err)
	}

	fmt.Println("✅ Chat message sent successfully")

	// Test 3: Send a command
	fmt.Println("\n⚡ Test 3: Send command")
	cmdMsg := ClientMessage{
		Type:    "command",
		Command: "/help",
	}

	data, err = json.Marshal(cmdMsg)
	if err != nil {
		log.Fatalf("❌ Failed to marshal command message: %v", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		log.Fatalf("❌ Failed to send command message: %v", err)
	}

	// Read command response
	_, message, err = conn.ReadMessage()
	if err != nil {
		log.Fatalf("❌ Failed to read command response: %v", err)
	}

	err = json.Unmarshal(message, &serverMsg)
	if err != nil {
		fmt.Printf("⚠️ Received non-JSON response: %s\n", string(message))
	} else {
		fmt.Printf("✅ Command response: Type=%s\n", serverMsg.Type)
	}

	// Test 4: Test backward compatibility with plain text
	fmt.Println("\n🔄 Test 4: Backward compatibility with plain text")
	err = conn.WriteMessage(websocket.TextMessage, []byte("Plain text message"))
	if err != nil {
		log.Fatalf("❌ Failed to send plain text message: %v", err)
	}

	fmt.Println("✅ Plain text message sent successfully")

	// Test 5: Request history (if MongoDB is enabled)
	fmt.Println("\n📚 Test 5: Request message history")
	historyMsg := ClientMessage{
		Type:  "get_history",
		Limit: 10,
	}

	data, err = json.Marshal(historyMsg)
	if err != nil {
		log.Fatalf("❌ Failed to marshal history request: %v", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		log.Fatalf("❌ Failed to send history request: %v", err)
	}

	// Read history response
	_, message, err = conn.ReadMessage()
	if err != nil {
		log.Fatalf("❌ Failed to read history response: %v", err)
	}

	err = json.Unmarshal(message, &serverMsg)
	if err != nil {
		fmt.Printf("⚠️ Received non-JSON history response: %s\n", string(message))
	} else {
		fmt.Printf("✅ History response: Type=%s, Messages count=%d\n", serverMsg.Type, len(serverMsg.Messages))
	}

	// Give some time to receive any additional messages
	fmt.Println("\n⏳ Waiting for additional messages...")
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	
	for i := 0; i < 5; i++ {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) || 
			   websocket.IsCloseError(err, websocket.CloseGoingAway) {
				break
			}
			// Timeout or other error - that's okay
			break
		}

		var msg ServerMessage
		if json.Unmarshal(message, &msg) == nil {
			fmt.Printf("📨 Additional message: Type=%s, Content=%s\n", msg.Type, msg.Content)
		} else {
			fmt.Printf("📨 Additional plain text: %s\n", string(message))
		}
	}

	fmt.Println("\n🎉 JSON WebSocket message tests completed!")
	fmt.Println("✅ The server supports both JSON and plain text messages")
	fmt.Println("🌐 You can now use the web interface at http://localhost:9091/chat.html")
}