package main

import (
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	fmt.Println("🧪 Testing /health Command and Heartbeat System")
	fmt.Println("===============================================")

	// เชื่อมต่อ
	u := url.URL{Scheme: "ws", Host: "localhost:9090", Path: "/ws"}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	fmt.Println("✅ Connected to server")

	// ตั้งค่า pong handler เพื่อตอบสนองต่อ ping
	conn.SetPongHandler(func(appData string) error {
		fmt.Println("💓 Received ping, sending pong response")
		return nil
	})

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

	// รอให้เซิร์ฟเวอร์ส่ง heartbeat ping
	fmt.Println("\n⏳ Waiting for heartbeat ping from server...")
	
	// ตั้งค่า read deadline เพื่อรอ ping
	conn.SetReadDeadline(time.Now().Add(35 * time.Second))
	
	// อ่านข้อความต่อไป (อาจเป็น ping หรือข้อความอื่น)
	for i := 0; i < 3; i++ {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway) {
				fmt.Println("🔌 Connection closed by server")
				break
			}
			fmt.Printf("⚠️ Read error: %v\n", err)
			break
		}
		
		if messageType == websocket.PingMessage {
			fmt.Println("💓 Received heartbeat ping from server!")
		} else {
			fmt.Printf("📨 Received message: %s\n", string(message))
		}
		
		// รอสักครู่ก่อนอ่านข้อความถัดไป
		time.Sleep(2 * time.Second)
	}

	// ส่งคำสั่ง /health อีกครั้งเพื่อดูการเปลี่ยนแปลง
	fmt.Println("\n💓 Sending /health command again to see updated stats...")
	err = conn.WriteMessage(websocket.TextMessage, []byte("/health"))
	if err != nil {
		log.Fatalf("Failed to send second /health command: %v", err)
	}

	// รับผลลัพธ์ครั้งที่สอง
	_, healthResponse2, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("Failed to read second health response: %v", err)
	}

	fmt.Println("💓 Updated Connection Health Status:")
	fmt.Println("===================================")
	fmt.Printf("%s\n", string(healthResponse2))

	fmt.Println("✅ /health command and heartbeat test completed!")
}