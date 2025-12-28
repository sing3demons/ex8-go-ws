package main

import (
	"fmt"
	"log"
	"net/url"

	"github.com/gorilla/websocket"
)

func main() {
	fmt.Println("🧪 Testing Configuration Management")
	fmt.Println("==================================")

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

	// ใช้ username "admin" เพื่อทดสอบ config commands
	adminUsername := "admin"
	err = conn.WriteMessage(websocket.TextMessage, []byte(adminUsername))
	if err != nil {
		log.Fatalf("Failed to send admin username: %v", err)
	}

	// รับข้อความต้อนรับ
	_, welcomeMsg, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("Failed to read welcome message: %v", err)
	}
	fmt.Printf("🎉 Received: %s\n", string(welcomeMsg))

	// ทดสอบ /config show command
	fmt.Println("\n⚙️ Testing /config show command...")
	err = conn.WriteMessage(websocket.TextMessage, []byte("/config show"))
	if err != nil {
		log.Fatalf("Failed to send /config show command: %v", err)
	}

	// รับผลลัพธ์
	_, configResponse, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("Failed to read config response: %v", err)
	}
	fmt.Printf("⚙️ Current Configuration:\n%s\n", string(configResponse))

	// ทดสอบ /config set command - เปลี่ยน max_message_length
	fmt.Println("\n🔧 Testing /config set command...")
	err = conn.WriteMessage(websocket.TextMessage, []byte("/config set max_message_length 500"))
	if err != nil {
		log.Fatalf("Failed to send /config set command: %v", err)
	}

	// รับผลลัพธ์
	_, setResponse, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("Failed to read set response: %v", err)
	}
	fmt.Printf("🔧 Set response: %s\n", string(setResponse))

	// ตรวจสอบการเปลี่ยนแปลง
	fmt.Println("\n🔍 Verifying configuration change...")
	err = conn.WriteMessage(websocket.TextMessage, []byte("/config show"))
	if err != nil {
		log.Fatalf("Failed to send verification command: %v", err)
	}

	// รับผลลัพธ์
	_, verifyResponse, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("Failed to read verify response: %v", err)
	}
	fmt.Printf("🔍 Updated Configuration:\n%s\n", string(verifyResponse))

	// ทดสอบ boolean setting
	fmt.Println("\n🎛️ Testing boolean configuration...")
	err = conn.WriteMessage(websocket.TextMessage, []byte("/config set enable_rate_limit false"))
	if err != nil {
		log.Fatalf("Failed to send boolean config: %v", err)
	}

	// รับผลลัพธ์
	_, boolResponse, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("Failed to read boolean response: %v", err)
	}
	fmt.Printf("🎛️ Boolean response: %s\n", string(boolResponse))

	// ทดสอบ duration setting
	fmt.Println("\n⏱️ Testing duration configuration...")
	err = conn.WriteMessage(websocket.TextMessage, []byte("/config set heartbeat_interval 45s"))
	if err != nil {
		log.Fatalf("Failed to send duration config: %v", err)
	}

	// รับผลลัพธ์
	_, durationResponse, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("Failed to read duration response: %v", err)
	}
	fmt.Printf("⏱️ Duration response: %s\n", string(durationResponse))

	// ทดสอบ invalid key
	fmt.Println("\n❌ Testing invalid configuration key...")
	err = conn.WriteMessage(websocket.TextMessage, []byte("/config set invalid_key 123"))
	if err != nil {
		log.Fatalf("Failed to send invalid config: %v", err)
	}

	// รับผลลัพธ์
	_, invalidResponse, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("Failed to read invalid response: %v", err)
	}
	fmt.Printf("❌ Invalid key response: %s\n", string(invalidResponse))

	// แสดงการตั้งค่าสุดท้าย
	fmt.Println("\n📋 Final configuration check...")
	err = conn.WriteMessage(websocket.TextMessage, []byte("/config show"))
	if err != nil {
		log.Fatalf("Failed to send final config check: %v", err)
	}

	// รับผลลัพธ์
	_, finalResponse, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("Failed to read final response: %v", err)
	}
	fmt.Printf("📋 Final Configuration:\n%s\n", string(finalResponse))

	fmt.Println("✅ Configuration management testing completed!")
}