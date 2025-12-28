package main

import (
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	fmt.Println("🧪 Testing Security Features")
	fmt.Println("============================")

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

	// ทดสอบ username validation - ใช้ชื่อที่มี HTML
	fmt.Println("\n🔒 Testing username validation...")
	invalidUsername := "<script>alert('xss')</script>"
	err = conn.WriteMessage(websocket.TextMessage, []byte(invalidUsername))
	if err != nil {
		log.Fatalf("Failed to send invalid username: %v", err)
	}

	// รับ error message
	_, errorMsg, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("Failed to read error message: %v", err)
	}
	fmt.Printf("🚫 Received error: %s\n", string(errorMsg))

	// ใช้ username ที่ถูกต้อง
	validUsername := "SecurityTestUser"
	err = conn.WriteMessage(websocket.TextMessage, []byte(validUsername))
	if err != nil {
		log.Fatalf("Failed to send valid username: %v", err)
	}

	// รับข้อความต้อนรับ
	_, welcomeMsg, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("Failed to read welcome message: %v", err)
	}
	fmt.Printf("🎉 Received: %s\n", string(welcomeMsg))

	// ทดสอบ /ratelimit command
	fmt.Println("\n⚡ Testing /ratelimit command...")
	err = conn.WriteMessage(websocket.TextMessage, []byte("/ratelimit"))
	if err != nil {
		log.Fatalf("Failed to send /ratelimit command: %v", err)
	}

	// รับผลลัพธ์
	_, rateLimitResponse, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("Failed to read rate limit response: %v", err)
	}
	fmt.Printf("⚡ Rate Limit Status:\n%s\n", string(rateLimitResponse))

	// ทดสอบ message validation - ข้อความยาวเกินไป
	fmt.Println("\n📝 Testing message validation...")
	longMessage := ""
	for i := 0; i < 1100; i++ { // เกิน MaxMessageLength (1000)
		longMessage += "a"
	}
	
	err = conn.WriteMessage(websocket.TextMessage, []byte(longMessage))
	if err != nil {
		log.Fatalf("Failed to send long message: %v", err)
	}

	// รับ error message
	_, longMsgError, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("Failed to read long message error: %v", err)
	}
	fmt.Printf("🚫 Long message error: %s\n", string(longMsgError))

	// ทดสอบ HTML sanitization
	fmt.Println("\n🛡️ Testing HTML sanitization...")
	htmlMessage := "<b>Bold text</b> and <script>alert('xss')</script>"
	err = conn.WriteMessage(websocket.TextMessage, []byte(htmlMessage))
	if err != nil {
		log.Fatalf("Failed to send HTML message: %v", err)
	}

	// รับข้อความที่ถูก sanitize (หรือ error)
	_, htmlResponse, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("Failed to read HTML response: %v", err)
	}
	fmt.Printf("🛡️ HTML response: %s\n", string(htmlResponse))

	// ทดสอบ rate limiting - ส่งข้อความเยอะๆ
	fmt.Println("\n🚀 Testing rate limiting...")
	for i := 1; i <= 12; i++ { // เกิน RateLimitMessages (10)
		message := fmt.Sprintf("Test message %d", i)
		err = conn.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Fatalf("Failed to send test message %d: %v", i, err)
		}

		// รับ response
		_, response, err := conn.ReadMessage()
		if err != nil {
			log.Fatalf("Failed to read response for message %d: %v", i, err)
		}
		
		if i <= 10 {
			fmt.Printf("✅ Message %d sent successfully\n", i)
		} else {
			fmt.Printf("🚫 Message %d blocked: %s\n", i, string(response))
		}
		
		time.Sleep(100 * time.Millisecond) // หน่วงเวลาเล็กน้อย
	}

	// ทดสอบ room name validation
	fmt.Println("\n🏠 Testing room name validation...")
	invalidRoomName := "invalid room name with spaces"
	err = conn.WriteMessage(websocket.TextMessage, []byte("/create "+invalidRoomName))
	if err != nil {
		log.Fatalf("Failed to send invalid room creation: %v", err)
	}

	// รับ error message
	_, roomError, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("Failed to read room error: %v", err)
	}
	fmt.Printf("🚫 Room creation error: %s\n", string(roomError))

	fmt.Println("\n✅ Security testing completed!")
}