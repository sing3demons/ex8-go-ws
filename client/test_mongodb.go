package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	fmt.Println("🧪 MongoDB Integration Test Client")
	fmt.Println("==================================")
	fmt.Println("This client tests MongoDB integration features:")
	fmt.Println("- Message persistence")
	fmt.Println("- Message history (/history)")
	fmt.Println("- Message search (/search)")
	fmt.Println("- User message history (/myhistory)")
	fmt.Println()

	// Connect to WebSocket server
	u := url.URL{Scheme: "ws", Host: "localhost:9091", Path: "/ws"}
	fmt.Printf("🔗 Connecting to %s\n", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("❌ Connection failed:", err)
	}
	defer conn.Close()

	fmt.Println("✅ Connected to server")

	// Handle incoming messages
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("❌ WebSocket error: %v", err)
				}
				return
			}
			fmt.Printf("📨 %s\n", message)
		}
	}()

	// Handle graceful shutdown
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Send authentication
	username := fmt.Sprintf("mongo_test_%d", time.Now().Unix()%1000)
	
	err = conn.WriteMessage(websocket.TextMessage, []byte(username))
	if err != nil {
		log.Printf("❌ Failed to send auth: %v", err)
		return
	}

	fmt.Printf("🔐 Authenticated as: %s\n", username)
	time.Sleep(1 * time.Second)

	// Auto-test sequence
	fmt.Println("\n🤖 Starting automated MongoDB test sequence...")
	
	testCommands := []struct {
		command     string
		description string
		delay       time.Duration
	}{
		{"/join general", "Join general room", 1 * time.Second},
		{"Hello from MongoDB test client!", "Send test message 1", 1 * time.Second},
		{"This is a test message for MongoDB persistence", "Send test message 2", 1 * time.Second},
		{"Testing message history functionality", "Send test message 3", 1 * time.Second},
		{"/history 5", "Get message history", 2 * time.Second},
		{"/search test", "Search for 'test' messages", 2 * time.Second},
		{"/myhistory 10", "Get my message history", 2 * time.Second},
		{"/create mongodb_test_room", "Create test room", 1 * time.Second},
		{"MongoDB integration test message in new room", "Send message in new room", 1 * time.Second},
		{"/history 3", "Get history in new room", 2 * time.Second},
		{"/join general", "Return to general room", 1 * time.Second},
		{"/search MongoDB", "Search for 'MongoDB' messages", 2 * time.Second},
	}

	// Execute test commands
	for i, test := range testCommands {
		fmt.Printf("\n📝 Test %d/%d: %s\n", i+1, len(testCommands), test.description)
		fmt.Printf("💬 Sending: %s\n", test.command)
		
		var message []byte
		if strings.HasPrefix(test.command, "/") {
			// Command
			message = []byte(test.command)
		} else {
			// Chat message
			message = []byte(test.command)
		}
		
		err = conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("❌ Failed to send message: %v", err)
			continue
		}
		
		time.Sleep(test.delay)
	}

	fmt.Println("\n✅ Automated test sequence completed!")
	fmt.Println("\n📝 Manual testing mode - Type commands or messages:")
	fmt.Println("Commands to try:")
	fmt.Println("  /history [limit]     - Show message history")
	fmt.Println("  /search <query>      - Search messages")
	fmt.Println("  /myhistory [limit]   - Show your message history")
	fmt.Println("  /help               - Show all commands")
	fmt.Println("  Type 'quit' to exit")

	// Manual input mode
	scanner := bufio.NewScanner(os.Stdin)
	
	for {
		select {
		case <-interrupt:
			fmt.Println("\n👋 Shutting down...")
			
			// Send close message
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Printf("❌ Error sending close message: %v", err)
			}
			
			// Wait for close or timeout
			select {
			case <-time.After(1 * time.Second):
			}
			return
			
		default:
			if scanner.Scan() {
				input := strings.TrimSpace(scanner.Text())
				
				if input == "quit" || input == "exit" {
					fmt.Println("👋 Goodbye!")
					return
				}
				
				if input == "" {
					continue
				}
				
				var message []byte
				if strings.HasPrefix(input, "/") {
					// Command
					message = []byte(input)
				} else {
					// Chat message
					message = []byte(input)
				}
				
				err = conn.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					log.Printf("❌ Failed to send message: %v", err)
					return
				}
			}
		}
	}
}