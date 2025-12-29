package main
import (
"fmt"
"log"
"time"
"github.com/gorilla/websocket"
)
func main() {
conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:9091/ws", nil)
if err != nil { log.Fatal(err) }
defer conn.Close()

go func() {
for {
_, message, err := conn.ReadMessage()
if err != nil { return }
fmt.Printf("📨 %s
", string(message))
}
}()

username := fmt.Sprintf("test%d", time.Now().Unix())
fmt.Printf("💬 Username: %s
", username)
conn.WriteMessage(websocket.TextMessage, []byte(username))
time.Sleep(3 * time.Second)

conn.WriteMessage(websocket.TextMessage, []byte("Hello World"))
time.Sleep(2 * time.Second)

conn.WriteMessage(websocket.TextMessage, []byte("/users"))
time.Sleep(3 * time.Second)
}
