package main

import (
	"fmt"
	"log"
	"net/url"

	"github.com/gorilla/websocket"
)

func main() {
	fmt.Println("🧪 Testing /health Command")
	fmt.Println("==========================")

	// เชื่อมต่อ
	u := url.URL{Scheme: "ws", Host: "localhost:9090", Path: "/ws"}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	fmt.Println("✅ Connected to server")

	// รับข้อความขอ username
	_, authMsg, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("Failed to read auth message: %v", err)
	}
	fmt.Printf("📨 Received: %s\n", string(authMsg))

	// ส่ง username
	username := "HealthTestUser"
	err = conn.WriteMessage(websocket.TextMessage, []byte(username))
	if err != nil {
		log.Fatalf("Failed to send username: %v", err)
	}
	fmt.Printf("📤 Sent username: %s\n", username)

	// รับข้อความต้อนรับ
	_, welcomeMsg, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("Failed to read welcome message: %v", err)
	}
	fmt.Printf("🎉 Received: %s\n", string(welcomeMsg))

	// ส่งคำสั่ง /health
	fmt.Println("\n💓 Sending /health command...")
	err = conn.WriteMessage(websocket.TextMessage, []byte("/health"))
	if err != nil {
		log.Fatalf("Failed to send /health command: %v", err)
	}

	// รับผลลัพธ์
	_, healthResponse, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("Failed to read health response: %v", err)
	}

	fmt.Println("💓 Connection Health Status:")
	fmt.Println("============================")
	fmt.Printf("%s\n", string(healthResponse))

	fmt.Println("✅ /health command test completed!")
}