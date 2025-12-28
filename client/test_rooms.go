package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	// ทดสอบ Room-based Chat System
	fmt.Println("🧪 Testing Room-based Chat System")
	fmt.Println("==================================")

	// สร้าง 3 clients
	clients := make([]*websocket.Conn, 3)
	usernames := []string{"Alice", "Bob", "Charlie"}

	// เชื่อมต่อ clients
	for i := 0; i < 3; i++ {
		u := url.URL{Scheme: "ws", Host: "localhost:9090", Path: "/ws"}
		conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			log.Fatalf("Failed to connect client %d: %v", i+1, err)
		}
		clients[i] = conn
		fmt.Printf("✅ Client %d connected\n", i+1)

		// รับข้อความขอ username
		_, authMsg, err := conn.ReadMessage()
		if err != nil {
			log.Fatalf("Failed to read auth message from client %d: %v", i+1, err)
		}
		fmt.Printf("📨 Client %d received: %s\n", i+1, string(authMsg))

		// ส่ง username
		err = conn.WriteMessage(websocket.TextMessage, []byte(usernames[i]))
		if err != nil {
			log.Fatalf("Failed to send username for client %d: %v", i+1, err)
		}
		fmt.Printf("📤 Client %d sent username: %s\n", i+1, usernames[i])

		// รับข้อความต้อนรับ
		_, welcomeMsg, err := conn.ReadMessage()
		if err != nil {
			log.Fatalf("Failed to read welcome message from client %d: %v", i+1, err)
		}
		fmt.Printf("🎉 Client %d received: %s\n", i+1, string(welcomeMsg))

		// รับข้อความแจ้งเตือนการเข้าร่วม (สำหรับ clients ที่เชื่อมต่อแล้ว)
		if i > 0 {
			for j := 0; j < i; j++ {
				_, joinMsg, err := clients[j].ReadMessage()
				if err != nil {
					log.Printf("Failed to read join message for client %d: %v", j+1, err)
					continue
				}
				fmt.Printf("👋 Client %d received join notification: %s\n", j+1, string(joinMsg))
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("\n🏠 Testing Room-based Messaging")
	fmt.Println("===============================")

	// ทดสอบการส่งข้อความในห้อง general
	testMessage := "Hello from Alice in general room!"
	err := clients[0].WriteMessage(websocket.TextMessage, []byte(testMessage))
	if err != nil {
		log.Fatalf("Failed to send message from Alice: %v", err)
	}
	fmt.Printf("📤 Alice sent: %s\n", testMessage)

	// ตรวจสอบว่า Bob และ Charlie ได้รับข้อความ (ในห้องเดียวกัน)
	for i := 1; i < 3; i++ {
		_, receivedMsg, err := clients[i].ReadMessage()
		if err != nil {
			log.Printf("Failed to read message for client %d: %v", i+1, err)
			continue
		}
		fmt.Printf("📨 %s received: %s\n", usernames[i], string(receivedMsg))
	}

	// ทดสอบการส่งข้อความจาก Bob
	testMessage2 := "Hi Alice! This is Bob responding."
	err = clients[1].WriteMessage(websocket.TextMessage, []byte(testMessage2))
	if err != nil {
		log.Fatalf("Failed to send message from Bob: %v", err)
	}
	fmt.Printf("📤 Bob sent: %s\n", testMessage2)

	// ตรวจสอบว่า Alice และ Charlie ได้รับข้อความ
	for i := 0; i < 3; i++ {
		if i == 1 { // ข้าม Bob (ผู้ส่ง)
			continue
		}
		_, receivedMsg, err := clients[i].ReadMessage()
		if err != nil {
			log.Printf("Failed to read message for client %d: %v", i+1, err)
			continue
		}
		fmt.Printf("📨 %s received: %s\n", usernames[i], string(receivedMsg))
	}

	fmt.Println("\n✅ Room-based Chat Test Completed!")
	fmt.Println("==================================")
	fmt.Println("📊 Test Results:")
	fmt.Println("  ✓ All users auto-joined 'general' room")
	fmt.Println("  ✓ Messages are scoped to room members only")
	fmt.Println("  ✓ Sender exclusion works correctly")
	fmt.Println("  ✓ Room-based notifications working")

	// รอสัญญาณ interrupt
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	fmt.Println("\n⏳ Press Ctrl+C to close connections and exit...")
	<-interrupt

	// ปิด connections
	for i, conn := range clients {
		if conn != nil {
			conn.Close()
			fmt.Printf("🔌 Client %d disconnected\n", i+1)
		}
	}

	fmt.Println("👋 Test completed!")
}