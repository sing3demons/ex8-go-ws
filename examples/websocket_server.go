package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// ตัวแปรสำหรับ upgrade HTTP connection เป็น WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// อนุญาตให้ทุก origin เชื่อมต่อได้ (ใช้เฉพาะ demo)
		// ในการใช้งานจริงควรตรวจสอบ origin ให้ดี
		return true
	},
}

// ฟังก์ชันจัดการ WebSocket connection
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	fmt.Println("🔗 มีคนพยายามเชื่อมต่อ WebSocket...")
	
	// อัพเกรดจาก HTTP เป็น WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ เกิดข้อผิดพลาดในการ upgrade: %v", err)
		return
	}
	defer conn.Close() // ปิดการเชื่อมต่อเมื่อจบฟังก์ชัน
	
	fmt.Println("✅ เชื่อมต่อ WebSocket สำเร็จ!")
	
	// วนลูปรับข้อความจาก client
	for {
		// อ่านข้อความจาก client
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("❌ เกิดข้อผิดพลาดในการอ่านข้อความ: %v", err)
			break // ออกจากลูปเมื่อเกิดข้อผิดพลาด
		}
		
		fmt.Printf("📨 ได้รับข้อความ: %s\n", message)
		
		// ส่งข้อความกลับไป (Echo Server)
		response := fmt.Sprintf("🔄 Server ได้รับ: %s", message)
		err = conn.WriteMessage(messageType, []byte(response))
		if err != nil {
			log.Printf("❌ เกิดข้อผิดพลาดในการส่งข้อความ: %v", err)
			break
		}
		
		fmt.Printf("📤 ส่งข้อความกลับ: %s\n", response)
	}
	
	fmt.Println("🔌 การเชื่อมต่อ WebSocket ปิดแล้ว")
}

// ฟังก์ชันแสดงหน้าเว็บง่ายๆ สำหรับทดสอบ
func handleHome(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>WebSocket Test</title>
    <meta charset="UTF-8">
</head>
<body>
    <h1>🧪 ทดสอบ WebSocket</h1>
    <div>
        <input type="text" id="messageInput" placeholder="พิมพ์ข้อความ..." />
        <button onclick="sendMessage()">ส่งข้อความ</button>
        <button onclick="connect()">เชื่อมต่อ</button>
        <button onclick="disconnect()">ตัดการเชื่อมต่อ</button>
    </div>
    <div>
        <h3>📋 ข้อความ:</h3>
        <div id="messages"></div>
    </div>

    <script>
        let ws = null;
        
        function connect() {
            ws = new WebSocket('ws://localhost:8082/ws');
            
            ws.onopen = function() {
                addMessage('✅ เชื่อมต่อสำเร็จ!');
            };
            
            ws.onmessage = function(event) {
                addMessage('📨 ' + event.data);
            };
            
            ws.onclose = function() {
                addMessage('🔌 การเชื่อมต่อปิดแล้ว');
            };
            
            ws.onerror = function(error) {
                addMessage('❌ เกิดข้อผิดพลาด: ' + error);
            };
        }
        
        function disconnect() {
            if (ws) {
                ws.close();
            }
        }
        
        function sendMessage() {
            const input = document.getElementById('messageInput');
            if (ws && input.value) {
                ws.send(input.value);
                addMessage('📤 ส่ง: ' + input.value);
                input.value = '';
            }
        }
        
        function addMessage(message) {
            const messages = document.getElementById('messages');
            messages.innerHTML += '<div>' + message + '</div>';
            messages.scrollTop = messages.scrollHeight;
        }
        
        // เชื่อมต่ออัตโนมัติเมื่อโหลดหน้า
        window.onload = connect;
    </script>
</body>
</html>`
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func main() {
	fmt.Println("🚀 เริ่มต้น WebSocket Server...")
	
	// กำหนด route สำหรับหน้าเว็บทดสอบ
	http.HandleFunc("/", handleHome)
	
	// กำหนด route สำหรับ WebSocket
	http.HandleFunc("/ws", handleWebSocket)
	
	// เริ่มต้น server ที่ port 8082
	fmt.Println("🌐 Server กำลังทำงานที่ http://localhost:8082")
	fmt.Println("🔗 WebSocket endpoint: ws://localhost:8082/ws")
	fmt.Println("📖 เปิดเบราว์เซอร์ไปที่ http://localhost:8082 เพื่อทดสอบ")
	
	log.Fatal(http.ListenAndServe(":8082", nil))
}