package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// WebSocket upgrader สำหรับ upgrade HTTP connection เป็น WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// อนุญาตให้ทุก origin เชื่อมต่อได้ (สำหรับการพัฒนา)
		return true
	},
}

// handleWebSocket จัดการ WebSocket connections
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection เป็น WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	// Log การเชื่อมต่อใหม่
	clientAddr := conn.RemoteAddr().String()
	log.Printf("New WebSocket connection from: %s", clientAddr)

	// Loop สำหรับรับและประมวลผลข้อความ
	for {
		// อ่านข้อความจาก client
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			// Log การตัดการเชื่อมต่อ
			log.Printf("Connection closed from %s: %v", clientAddr, err)
			break
		}

		// แสดงข้อความที่ได้รับใน console
		log.Printf("Received from %s: %s", clientAddr, string(message))

		// ส่งข้อความ echo กลับไปยัง client
		echoMessage := "Echo: " + string(message)
		err = conn.WriteMessage(messageType, []byte(echoMessage))
		if err != nil {
			log.Printf("Failed to send echo message to %s: %v", clientAddr, err)
			break
		}

		log.Printf("Sent echo to %s: %s", clientAddr, echoMessage)
	}
}

func main() {
	// ตั้งค่า HTTP routes
	http.HandleFunc("/ws", handleWebSocket)
	
	// เสิร์ฟ static files สำหรับ test client (จะสร้างในขั้นตอนถัดไป)
	http.Handle("/", http.FileServer(http.Dir("./static/")))

	// เริ่มต้น server
	port := ":9090"
	log.Printf("Starting WebSocket server on port %s", port)
	log.Printf("WebSocket endpoint: ws://localhost%s/ws", port)
	log.Printf("Test page: http://localhost%s", port)
	
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal("Server failed to start:", err)
	}
}