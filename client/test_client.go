package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/gorilla/websocket"
)

func main() {
	// เชื่อมต่อไปยัง WebSocket server
	serverURL := "ws://localhost:9090/ws"
	fmt.Printf("🔄 กำลังเชื่อมต่อไปยัง %s...\n", serverURL)

	conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer conn.Close()

	fmt.Println("✅ เชื่อมต่อสำเร็จ!")
	fmt.Println("💬 พิมพ์ข้อความและกด Enter เพื่อส่ง (พิมพ์ 'quit' เพื่อออก)")

	// Channel สำหรับจัดการ graceful shutdown
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Goroutine สำหรับรับข้อความจาก server
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				fmt.Printf("❌ Error reading message: %v\n", err)
				return
			}
			fmt.Printf("📨 Server: %s\n", string(message))
		}
	}()

	// อ่าน input จาก user
	scanner := bufio.NewScanner(os.Stdin)
	for {
		select {
		case <-interrupt:
			fmt.Println("\n👋 กำลังปิดการเชื่อมต่อ...")
			return
		default:
			fmt.Print("💬 You: ")
			if scanner.Scan() {
				message := strings.TrimSpace(scanner.Text())
				
				if message == "quit" {
					fmt.Println("👋 ลาก่อน!")
					return
				}
				
				if message != "" {
					err := conn.WriteMessage(websocket.TextMessage, []byte(message))
					if err != nil {
						fmt.Printf("❌ Error sending message: %v\n", err)
						return
					}
				}
			}
		}
	}
}