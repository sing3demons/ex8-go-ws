package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// โครงสร้างข้อความแชท
type ChatMessage struct {
	Type      string    `json:"type"`      // ประเภทข้อความ: "join", "leave", "message", "userCount"
	Username  string    `json:"username"`  // ชื่อผู้ใช้
	Message   string    `json:"message"`   // ข้อความ
	Timestamp time.Time `json:"timestamp"` // เวลาที่ส่ง
	UserCount int       `json:"userCount"` // จำนวนผู้ใช้ออนไลน์
}

// โครงสร้างผู้ใช้
type Client struct {
	conn     *websocket.Conn  // การเชื่อมต่อ WebSocket
	username string           // ชื่อผู้ใช้
	send     chan ChatMessage // channel สำหรับส่งข้อความ
}

// โครงสร้าง Chat Hub - จัดการผู้ใช้ทั้งหมด
type ChatHub struct {
	clients    map[*Client]bool // รายชื่อผู้ใช้ที่เชื่อมต่อ
	broadcast  chan ChatMessage // channel สำหรับส่งข้อความให้ทุกคน
	register   chan *Client     // channel สำหรับลงทะเบียนผู้ใช้ใหม่
	unregister chan *Client     // channel สำหรับยกเลิกการลงทะเบียน
	mutex      sync.RWMutex     // mutex สำหรับ thread safety
}

// สร้าง Chat Hub ใหม่
func newChatHub() *ChatHub {
	return &ChatHub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan ChatMessage, 100), // เพิ่ม buffer size
		register:   make(chan *Client, 10),      // เพิ่ม buffer size
		unregister: make(chan *Client, 10),      // เพิ่ม buffer size
	}
}

// ส่งข้อความอัพเดท user count ให้ทุกคน
func (h *ChatHub) broadcastUserCount() {
	h.mutex.RLock()
	userCount := len(h.clients)
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mutex.RUnlock()

	countMessage := ChatMessage{
		Type:      "userCount",
		Username:  "ระบบ",
		Message:   "",
		Timestamp: time.Now(),
		UserCount: userCount,
	}

	// ส่งข้อความให้ทุกคนโดยไม่ถือ lock
	var failedClients []*Client
	for _, client := range clients {
		select {
		case client.send <- countMessage:
			// ส่งสำเร็จ
		default:
			// ส่งไม่ได้ เก็บไว้ลบทีหลัง
			failedClients = append(failedClients, client)
		}
	}

	// ลบ clients ที่ส่งไม่ได้
	if len(failedClients) > 0 {
		h.mutex.Lock()
		for _, client := range failedClients {
			if _, ok := h.clients[client]; ok {
				close(client.send)
				delete(h.clients, client)
			}
		}
		h.mutex.Unlock()
	}
}

// ฟังก์ชันหลักของ Chat Hub
func (h *ChatHub) run() {
	for {
		select {
		// มีผู้ใช้ใหม่เข้าร่วม
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			userCount := len(h.clients)
			h.mutex.Unlock()

			fmt.Printf("✅ %s เข้าร่วมแชท (ผู้ใช้ออนไลน์: %d คน)\n", client.username, userCount)

			// ส่งข้อความต้อนรับให้ผู้ใช้ใหม่
			welcomeMessage := ChatMessage{
				Type:      "welcome",
				Username:  "ระบบ",
				Message:   fmt.Sprintf("ยินดีต้อนรับ %s! 🎉", client.username),
				Timestamp: time.Now(),
				UserCount: userCount,
			}
			
			// ส่งข้อความต้อนรับโดยไม่บล็อก
			go func() {
				select {
				case client.send <- welcomeMessage:
				case <-time.After(5 * time.Second):
					// timeout หากส่งไม่ได้ภายใน 5 วินาที
					fmt.Printf("⚠️ ไม่สามารถส่งข้อความต้อนรับให้ %s ได้\n", client.username)
				}
			}()

			// ส่งข้อความแจ้งให้ทุกคนทราบ
			joinMessage := ChatMessage{
				Type:      "join",
				Username:  "ระบบ",
				Message:   fmt.Sprintf("🎉 %s เข้าร่วมแชท", client.username),
				Timestamp: time.Now(),
				UserCount: userCount,
			}
			
			// ส่งข้อความ join โดยไม่บล็อก
			go func() {
				select {
				case h.broadcast <- joinMessage:
				case <-time.After(5 * time.Second):
					fmt.Printf("⚠️ ไม่สามารถส่งข้อความ join สำหรับ %s ได้\n", client.username)
				}
			}()

			// อัพเดท user count ให้ทุกคน
			go h.broadcastUserCount()

		// มีผู้ใช้ออกจากแชท
		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				userCount := len(h.clients)
				h.mutex.Unlock()

				fmt.Printf("❌ %s ออกจากแชท (ผู้ใช้ออนไลน์: %d คน)\n", client.username, userCount)

				// ส่งข้อความแจ้งให้ทุกคนทราบ
				leaveMessage := ChatMessage{
					Type:      "leave",
					Username:  "ระบบ",
					Message:   fmt.Sprintf("👋 %s ออกจากแชท", client.username),
					Timestamp: time.Now(),
					UserCount: userCount,
				}
				
				// ส่งข้อความ leave โดยไม่บล็อก
				go func() {
					select {
					case h.broadcast <- leaveMessage:
					case <-time.After(5 * time.Second):
						fmt.Printf("⚠️ ไม่สามารถส่งข้อความ leave สำหรับ %s ได้\n", client.username)
					}
				}()

				// อัพเดท user count ให้ทุกคน
				go h.broadcastUserCount()
			} else {
				h.mutex.Unlock()
			}

		// มีข้อความใหม่ที่ต้องส่งให้ทุกคน
		case message := <-h.broadcast:
			h.mutex.RLock()
			clients := make([]*Client, 0, len(h.clients))
			for client := range h.clients {
				clients = append(clients, client)
			}
			h.mutex.RUnlock()

			// ส่งข้อความให้ทุกคนโดยไม่ถือ lock
			var failedClients []*Client
			for _, client := range clients {
				select {
				case client.send <- message:
					// ส่งสำเร็จ
				default:
					// ส่งไม่ได้ เก็บไว้ลบทีหลัง
					failedClients = append(failedClients, client)
				}
			}

			// ลบ clients ที่ส่งไม่ได้
			if len(failedClients) > 0 {
				h.mutex.Lock()
				for _, client := range failedClients {
					if _, ok := h.clients[client]; ok {
						close(client.send)
						delete(h.clients, client)
					}
				}
				h.mutex.Unlock()
			}
		}
	}
}

// ตัวแปรสำหรับ upgrade HTTP เป็น WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // อนุญาตทุก origin (ใช้เฉพาะ demo)
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// ฟังก์ชันอ่านข้อความจาก client
func (c *Client) readPump(hub *ChatHub) {
	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()

	// ตั้งค่า timeout และ ping/pong
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg ChatMessage
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// เพิ่มข้อมูลเพิ่มเติม
		msg.Username = c.username
		msg.Timestamp = time.Now()
		msg.Type = "message"

		// ส่งข้อความให้ทุกคน
		select {
		case hub.broadcast <- msg:
		default:
			// ถ้า broadcast channel เต็ม ให้ข้าม
		}
	}
}

// ฟังก์ชันส่งข้อความไปยัง client
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				log.Println(err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ตัวแปร global สำหรับ chat hub
var chatHub = newChatHub()

// ฟังก์ชันจัดการ WebSocket connection
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// ดึงชื่อผู้ใช้จาก query parameter
	username := r.URL.Query().Get("username")
	if username == "" {
		username = fmt.Sprintf("ผู้ใช้%d", time.Now().Unix()%1000)
	}

	fmt.Printf("🔗 มีคนพยายามเชื่อมต่อ: %s\n", username)

	// อัพเกรดเป็น WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade error: %v", err)
		return
	}

	fmt.Printf("✅ WebSocket upgrade สำเร็จสำหรับ: %s\n", username)

	// สร้าง client ใหม่
	client := &Client{
		conn:     conn,
		username: username,
		send:     make(chan ChatMessage, 512), // เพิ่ม buffer size
	}

	// ลงทะเบียน client
	fmt.Printf("📝 กำลังลงทะเบียน client: %s\n", username)
	chatHub.register <- client

	// เริ่ม goroutines สำหรับอ่านและเขียน
	go client.writePump()
	go client.readPump(chatHub)
}

// ฟังก์ชันแสดงหน้าเว็บแชท
func handleChatPage(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>💬 Chat Room</title>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f0f0f0; }
        .container { max-width: 800px; margin: 0 auto; background: white; border-radius: 10px; padding: 20px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { text-align: center; color: #333; border-bottom: 2px solid #007bff; padding-bottom: 10px; margin-bottom: 20px; }
        .chat-container { height: 400px; border: 1px solid #ddd; border-radius: 5px; overflow-y: auto; padding: 10px; background: #fafafa; margin-bottom: 10px; }
        .message { margin: 5px 0; padding: 8px; border-radius: 5px; }
        .message.system { background: #e7f3ff; color: #0066cc; font-style: italic; }
        .message.welcome { background: #d4edda; color: #155724; font-weight: bold; }
        .message.user { background: #e8f5e8; }
        .message.other { background: #fff3cd; }
        .input-container { display: flex; gap: 10px; }
        .input-container input { flex: 1; padding: 10px; border: 1px solid #ddd; border-radius: 5px; }
        .input-container button { padding: 10px 20px; background: #007bff; color: white; border: none; border-radius: 5px; cursor: pointer; }
        .input-container button:hover { background: #0056b3; }
        .status { text-align: center; color: #666; margin: 10px 0; }
        .username-input { margin-bottom: 20px; text-align: center; }
        .username-input input { padding: 10px; margin: 0 10px; border: 1px solid #ddd; border-radius: 5px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>💬 Chat Room แบบ Real-time</h1>
            <div class="status" id="status">🔴 ไม่ได้เชื่อมต่อ</div>
            <div class="status" id="userCount">👥 ผู้ใช้ออนไลน์: 0 คน</div>
        </div>
        
        <div class="username-input">
            <label>ชื่อของคุณ:</label>
            <input type="text" id="usernameInput" placeholder="พิมพ์ชื่อ..." value="">
            <button onclick="connect()">เข้าร่วมแชท</button>
            <button onclick="disconnect()">ออกจากแชท</button>
        </div>
        
        <div class="chat-container" id="chatContainer"></div>
        
        <div class="input-container">
            <input type="text" id="messageInput" placeholder="พิมพ์ข้อความ..." disabled>
            <button onclick="sendMessage()" id="sendButton" disabled>ส่ง</button>
        </div>
    </div>

    <script>
        let ws = null;
        let username = '';
        
        function connect() {
            const usernameInput = document.getElementById('usernameInput');
            username = usernameInput.value.trim() || 'ผู้ใช้' + Math.floor(Math.random()*1000);
            
            ws = new WebSocket('ws://localhost:8083/ws?username=' + encodeURIComponent(username));
            
            ws.onopen = function() {
                updateStatus('🟢 เชื่อมต่อแล้ว', 'green');
                document.getElementById('messageInput').disabled = false;
                document.getElementById('sendButton').disabled = false;
                usernameInput.disabled = true;
            };
            
            ws.onmessage = function(event) {
                const message = JSON.parse(event.data);
                displayMessage(message);
                updateUserCount(message.userCount);
            };
            
            ws.onclose = function() {
                updateStatus('🔴 การเชื่อมต่อปิดแล้ว', 'red');
                document.getElementById('messageInput').disabled = true;
                document.getElementById('sendButton').disabled = true;
                document.getElementById('usernameInput').disabled = false;
            };
            
            ws.onerror = function(error) {
                updateStatus('❌ เกิดข้อผิดพลาด', 'red');
                console.error('WebSocket error:', error);
            };
        }
        
        function disconnect() {
            if (ws) {
                ws.close();
            }
        }
        
        function sendMessage() {
            const input = document.getElementById('messageInput');
            if (ws && input.value.trim()) {
                const message = {
                    message: input.value.trim()
                };
                ws.send(JSON.stringify(message));
                input.value = '';
            }
        }
        
        function displayMessage(msg) {
            const container = document.getElementById('chatContainer');
            const messageDiv = document.createElement('div');
            messageDiv.className = 'message';
            
            const time = new Date(msg.timestamp).toLocaleTimeString('th-TH');
            
            if (msg.type === 'join' || msg.type === 'leave') {
                messageDiv.className += ' system';
                messageDiv.innerHTML = '<strong>' + time + '</strong> ' + msg.message;
            } else if (msg.type === 'welcome') {
                messageDiv.className += ' welcome';
                messageDiv.innerHTML = '<strong>' + time + '</strong> ' + msg.message;
            } else if (msg.type === 'userCount') {
                // ไม่แสดงข้อความ userCount แต่อัพเดทเฉพาะตัวเลข
                return;
            } else {
                if (msg.username === username) {
                    messageDiv.className += ' user';
                    messageDiv.innerHTML = '<strong>' + time + ' คุณ:</strong> ' + msg.message;
                } else {
                    messageDiv.className += ' other';
                    messageDiv.innerHTML = '<strong>' + time + ' ' + msg.username + ':</strong> ' + msg.message;
                }
            }
            
            container.appendChild(messageDiv);
            container.scrollTop = container.scrollHeight;
        }
        
        function updateStatus(status, color) {
            const statusElement = document.getElementById('status');
            statusElement.textContent = status;
            statusElement.style.color = color;
        }
        
        function updateUserCount(count) {
            if (count !== undefined) {
                document.getElementById('userCount').textContent = '👥 ผู้ใช้ออนไลน์: ' + count + ' คน';
            }
        }
        
        // ส่งข้อความเมื่อกด Enter
        document.getElementById('messageInput').addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                sendMessage();
            }
        });
        
        // ส่งข้อความเมื่อกด Enter ในช่องชื่อ
        document.getElementById('usernameInput').addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                connect();
            }
        });
        
        // ตั้งชื่อเริ่มต้นเมื่อโหลดหน้า
        window.onload = function() {
            document.getElementById('usernameInput').value = 'ผู้ใช้' + Math.floor(Math.random()*1000);
        };
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func main() {
	fmt.Println("🚀 เริ่มต้น Chat Server (Fixed Version)...")

	// เริ่ม chat hub
	go chatHub.run()

	// กำหนด routes
	http.HandleFunc("/", handleChatPage)
	http.HandleFunc("/ws", handleWebSocket)

	// เริ่มต้น server
	fmt.Println("🌐 Chat Server กำลังทำงานที่ http://localhost:8083")
	fmt.Println("💬 เปิดเบราว์เซอร์หลายหน้าต่างเพื่อทดสอบแชทหลายคน!")

	log.Fatal(http.ListenAndServe(":8083", nil))
}
