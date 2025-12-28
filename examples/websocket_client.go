package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	fmt.Println("🚀 เริ่มต้น WebSocket Client...")
	
	// เชื่อมต่อไปยัง WebSocket server
	fmt.Println("🔗 กำลังเชื่อมต่อไปยัง ws://localhost:8080/ws...")
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		log.Fatal("❌ เชื่อมต่อไม่ได้:", err)
	}
	defer conn.Close()
	
	fmt.Println("✅ เชื่อมต่อสำเร็จ!")
	fmt.Println("💬 พิมพ์ข้อความแล้วกด Enter (พิมพ์ 'quit' เพื่อออก)")
	
	// สร้าง channel สำหรับรับข้อความจาก server
	done := make(chan struct{})
	
	// Goroutine สำหรับรับข้อความจาก server
	go func() {
		defer close(done)
		for {
			// อ่านข้อความจาก server
			_, message, err := conn.ReadMessage()
			if err != nil {
				fmt.Printf("❌ เกิดข้อผิดพลาดในการอ่านข้อความ: %v\n", err)
				return
			}
			fmt.Printf("📨 %s\n", message)
		}
	}()
	
	// อ่านข้อความจาก keyboard
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("💬 พิมพ์ข้อความ: ")
		if !scanner.Scan() {
			break
		}
		
		message := strings.TrimSpace(scanner.Text())
		
		// ตรวจสอบคำสั่งออก
		if message == "quit" || message == "exit" {
			fmt.Println("👋 ลาก่อน!")
			break
		}
		
		// ข้ามถ้าข้อความว่าง
		if message == "" {
			continue
		}
		
		// ส่งข้อความไปยัง server
		err := conn.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			fmt.Printf("❌ ส่งข้อความไม่ได้: %v\n", err)
			break
		}
		
		fmt.Printf("📤 ส่งแล้ว: %s\n", message)
	}
	
	// ปิดการเชื่อมต่อ
	err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		fmt.Printf("❌ ปิดการเชื่อมต่อไม่ได้: %v\n", err)
		return
	}
	
	// รอให้ server ปิดการเชื่อมต่อ
	select {
	case <-done:
	case <-time.After(time.Second):
	}
	
	fmt.Println("🔌 การเชื่อมต่อปิดแล้ว")
}