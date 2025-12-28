package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	fmt.Println("🧪 Testing Broadcasting System...")
	fmt.Println("📡 Creating multiple WebSocket connections...")

	var wg sync.WaitGroup
	numClients := 3

	// สร้าง multiple clients
	for i := 1; i <= numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			testClient(clientID)
		}(i)
	}

	// รอให้ clients เชื่อมต่อ
	time.Sleep(2 * time.Second)

	fmt.Println("\n✅ All clients connected. Broadcasting system is ready!")
	fmt.Println("💬 Each client will send a message and receive messages from others")
	fmt.Println("🔍 Check server logs to see the broadcasting in action")

	wg.Wait()
}

func testClient(clientID int) {
	serverURL := "ws://localhost:9090/ws"
	
	// เชื่อมต่อไปยัง server
	conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		log.Printf("❌ Client %d failed to connect: %v", clientID, err)
		return
	}
	defer conn.Close()

	fmt.Printf("✅ Client %d connected\n", clientID)

	// Goroutine สำหรับรับข้อความ
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			fmt.Printf("📨 Client %d received: %s\n", clientID, string(message))
		}
	}()

	// รอสักครู่แล้วส่งข้อความ
	time.Sleep(time.Duration(clientID) * time.Second)
	
	testMessage := fmt.Sprintf("Hello from Client %d! 👋", clientID)
	err = conn.WriteMessage(websocket.TextMessage, []byte(testMessage))
	if err != nil {
		log.Printf("❌ Client %d failed to send message: %v", clientID, err)
		return
	}

	fmt.Printf("📤 Client %d sent: %s\n", clientID, testMessage)

	// รอรับข้อความจาก clients อื่น
	time.Sleep(5 * time.Second)
}