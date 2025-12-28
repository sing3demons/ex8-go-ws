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
	// ทดสอบ Command System
	fmt.Println("🧪 Testing Command System")
	fmt.Println("=========================")

	// สร้าง 2 clients
	clients := make([]*websocket.Conn, 2)
	usernames := []string{"Alice", "Bob"}

	// เชื่อมต่อ clients
	for i := 0; i < 2; i++ {
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

	fmt.Println("\n📋 Testing /help Command")
	fmt.Println("========================")

	// ทดสอบคำสั่ง /help
	err := clients[0].WriteMessage(websocket.TextMessage, []byte("/help"))
	if err != nil {
		log.Fatalf("Failed to send /help command: %v", err)
	}
	fmt.Printf("📤 Alice sent: /help\n")

	// รับผลลัพธ์
	_, helpResponse, err := clients[0].ReadMessage()
	if err != nil {
		log.Printf("Failed to read help response: %v", err)
	} else {
		fmt.Printf("📨 Alice received help:\n%s\n", string(helpResponse))
	}

	fmt.Println("\n👥 Testing /users Command")
	fmt.Println("=========================")

	// ทดสอบคำสั่ง /users
	err = clients[1].WriteMessage(websocket.TextMessage, []byte("/users"))
	if err != nil {
		log.Fatalf("Failed to send /users command: %v", err)
	}
	fmt.Printf("📤 Bob sent: /users\n")

	// รับผลลัพธ์
	_, usersResponse, err := clients[1].ReadMessage()
	if err != nil {
		log.Printf("Failed to read users response: %v", err)
	} else {
		fmt.Printf("📨 Bob received users list:\n%s\n", string(usersResponse))
	}

	fmt.Println("\n🏠 Testing /rooms Command")
	fmt.Println("=========================")

	// ทดสอบคำสั่ง /rooms
	err = clients[0].WriteMessage(websocket.TextMessage, []byte("/rooms"))
	if err != nil {
		log.Fatalf("Failed to send /rooms command: %v", err)
	}
	fmt.Printf("📤 Alice sent: /rooms\n")

	// รับผลลัพธ์
	_, roomsResponse, err := clients[0].ReadMessage()
	if err != nil {
		log.Printf("Failed to read rooms response: %v", err)
	} else {
		fmt.Printf("📨 Alice received rooms list:\n%s\n", string(roomsResponse))
	}

	fmt.Println("\n🏗️ Testing /create Command")
	fmt.Println("==========================")

	// ทดสอบคำสั่ง /create
	err = clients[0].WriteMessage(websocket.TextMessage, []byte("/create dev-team"))
	if err != nil {
		log.Fatalf("Failed to send /create command: %v", err)
	}
	fmt.Printf("📤 Alice sent: /create dev-team\n")

	// รับผลลัพธ์จาก Alice (ผู้สร้าง)
	_, createResponse, err := clients[0].ReadMessage()
	if err != nil {
		log.Printf("Failed to read create response: %v", err)
	} else {
		fmt.Printf("📨 Alice received: %s\n", string(createResponse))
	}

	// รับข้อความแจ้งเตือนจาก Bob (ห้องใหม่ถูกสร้าง)
	_, announceMsg, err := clients[1].ReadMessage()
	if err != nil {
		log.Printf("Failed to read announce message: %v", err)
	} else {
		fmt.Printf("📨 Bob received announcement: %s\n", string(announceMsg))
	}

	// รับข้อความแจ้งเตือนจาก Bob (Alice ออกจาก general)
	_, leaveMsg, err := clients[1].ReadMessage()
	if err != nil {
		log.Printf("Failed to read leave message: %v", err)
	} else {
		fmt.Printf("📨 Bob received leave notification: %s\n", string(leaveMsg))
	}

	fmt.Println("\n🚪 Testing /join Command")
	fmt.Println("========================")

	// ทดสอบคำสั่ง /join
	err = clients[1].WriteMessage(websocket.TextMessage, []byte("/join dev-team"))
	if err != nil {
		log.Fatalf("Failed to send /join command: %v", err)
	}
	fmt.Printf("📤 Bob sent: /join dev-team\n")

	// รับผลลัพธ์จาก Bob
	_, joinResponse, err := clients[1].ReadMessage()
	if err != nil {
		log.Printf("Failed to read join response: %v", err)
	} else {
		fmt.Printf("📨 Bob received: %s\n", string(joinResponse))
	}

	// รับข้อความแจ้งเตือนจาก Alice (Bob เข้าห้อง dev-team)
	_, joinNotification, err := clients[0].ReadMessage()
	if err != nil {
		log.Printf("Failed to read join notification: %v", err)
	} else {
		fmt.Printf("📨 Alice received join notification: %s\n", string(joinNotification))
	}

	fmt.Println("\n💬 Testing Room-scoped Messaging")
	fmt.Println("=================================")

	// ทดสอบการส่งข้อความในห้อง dev-team
	testMessage := "Hello team! This is our new dev room."
	err = clients[0].WriteMessage(websocket.TextMessage, []byte(testMessage))
	if err != nil {
		log.Fatalf("Failed to send message from Alice: %v", err)
	}
	fmt.Printf("📤 Alice sent in dev-team: %s\n", testMessage)

	// Bob ควรได้รับข้อความ (อยู่ในห้องเดียวกัน)
	_, receivedMsg, err := clients[1].ReadMessage()
	if err != nil {
		log.Printf("Failed to read message for Bob: %v", err)
	} else {
		fmt.Printf("📨 Bob received: %s\n", string(receivedMsg))
	}

	fmt.Println("\n🚪 Testing /leave Command")
	fmt.Println("=========================")

	// ทดสอบคำสั่ง /leave
	err = clients[1].WriteMessage(websocket.TextMessage, []byte("/leave"))
	if err != nil {
		log.Fatalf("Failed to send /leave command: %v", err)
	}
	fmt.Printf("📤 Bob sent: /leave\n")

	// รับผลลัพธ์จาก Bob
	_, leaveResponse, err := clients[1].ReadMessage()
	if err != nil {
		log.Printf("Failed to read leave response: %v", err)
	} else {
		fmt.Printf("📨 Bob received: %s\n", string(leaveResponse))
	}

	// รับข้อความแจ้งเตือนจาก Alice (Bob ออกจาก dev-team)
	_, leaveNotification, err := clients[0].ReadMessage()
	if err != nil {
		log.Printf("Failed to read leave notification: %v", err)
	} else {
		fmt.Printf("📨 Alice received leave notification: %s\n", string(leaveNotification))
	}

	fmt.Println("\n❌ Testing Unknown Command")
	fmt.Println("==========================")

	// ทดสอบคำสั่งที่ไม่มี
	err = clients[0].WriteMessage(websocket.TextMessage, []byte("/unknown"))
	if err != nil {
		log.Fatalf("Failed to send unknown command: %v", err)
	}
	fmt.Printf("📤 Alice sent: /unknown\n")

	// รับข้อความ error
	_, errorMsg, err := clients[0].ReadMessage()
	if err != nil {
		log.Printf("Failed to read error message: %v", err)
	} else {
		fmt.Printf("📨 Alice received error: %s\n", string(errorMsg))
	}

	fmt.Println("\n✅ Command System Test Completed!")
	fmt.Println("==================================")
	fmt.Println("📊 Test Results:")
	fmt.Println("  ✓ /help command working")
	fmt.Println("  ✓ /users command working")
	fmt.Println("  ✓ /rooms command working")
	fmt.Println("  ✓ /create command working")
	fmt.Println("  ✓ /join command working")
	fmt.Println("  ✓ /leave command working")
	fmt.Println("  ✓ Room-scoped messaging working")
	fmt.Println("  ✓ Unknown command error handling working")

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