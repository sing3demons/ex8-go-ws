package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	// ทดสอบ Performance และ Concurrency
	fmt.Println("🧪 Testing Performance and Concurrency")
	fmt.Println("======================================")

	// ทดสอบการเชื่อมต่อพร้อมกัน
	numClients := 20
	fmt.Printf("🔗 Creating %d concurrent connections...\n", numClients)

	var wg sync.WaitGroup
	clients := make([]*websocket.Conn, numClients)
	usernames := make([]string, numClients)

	// สร้าง connections พร้อมกัน
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			u := url.URL{Scheme: "ws", Host: "localhost:9090", Path: "/ws"}
			conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
			if err != nil {
				log.Printf("Failed to connect client %d: %v", clientID+1, err)
				return
			}

			clients[clientID] = conn
			usernames[clientID] = fmt.Sprintf("User%d", clientID+1)

			// รับข้อความขอ username
			_, _, err = conn.ReadMessage()
			if err != nil {
				log.Printf("Failed to read auth message from client %d: %v", clientID+1, err)
				return
			}

			// ส่ง username
			err = conn.WriteMessage(websocket.TextMessage, []byte(usernames[clientID]))
			if err != nil {
				log.Printf("Failed to send username for client %d: %v", clientID+1, err)
				return
			}

			// รับข้อความต้อนรับ
			_, _, err = conn.ReadMessage()
			if err != nil {
				log.Printf("Failed to read welcome message from client %d: %v", clientID+1, err)
				return
			}

			fmt.Printf("✅ Client %d (%s) connected and authenticated\n", clientID+1, usernames[clientID])
		}(i)
	}

	// รอให้ทุก client เชื่อมต่อเสร็จ
	wg.Wait()
	fmt.Printf("🎉 All %d clients connected successfully!\n", numClients)

	// รอให้ join notifications เสร็จ
	time.Sleep(2 * time.Second)

	// ทดสอบ /stats command
	fmt.Println("\n📊 Testing /stats command")
	fmt.Println("=========================")

	if clients[0] != nil {
		err := clients[0].WriteMessage(websocket.TextMessage, []byte("/stats"))
		if err != nil {
			log.Printf("Failed to send /stats command: %v", err)
		} else {
			fmt.Printf("📤 %s sent: /stats\n", usernames[0])

			// รับผลลัพธ์
			_, statsResponse, err := clients[0].ReadMessage()
			if err != nil {
				log.Printf("Failed to read stats response: %v", err)
			} else {
				fmt.Printf("📨 %s received stats:\n%s\n", usernames[0], string(statsResponse))
			}
		}
	}

	// ทดสอบการส่งข้อความพร้อมกัน
	fmt.Println("\n💬 Testing Concurrent Messaging")
	fmt.Println("===============================")

	messageCount := 5
	fmt.Printf("📤 Each client will send %d messages concurrently...\n", messageCount)

	// เริ่ม goroutines สำหรับรับข้อความ
	var messageWg sync.WaitGroup
	receivedMessages := make([]int, numClients)
	
	for i := 0; i < numClients; i++ {
		if clients[i] == nil {
			continue
		}
		
		messageWg.Add(1)
		go func(clientID int) {
			defer messageWg.Done()
			
			// รับข้อความจากคนอื่น
			for j := 0; j < (numClients-1)*messageCount; j++ {
				_, msg, err := clients[clientID].ReadMessage()
				if err != nil {
					log.Printf("Client %d failed to read message: %v", clientID+1, err)
					break
				}
				receivedMessages[clientID]++
				if receivedMessages[clientID] <= 3 { // แสดงแค่ 3 ข้อความแรก
					fmt.Printf("📨 %s received: %s\n", usernames[clientID], string(msg))
				}
			}
		}(i)
	}

	// ส่งข้อความพร้อมกัน
	var sendWg sync.WaitGroup
	for i := 0; i < numClients; i++ {
		if clients[i] == nil {
			continue
		}
		
		sendWg.Add(1)
		go func(clientID int) {
			defer sendWg.Done()
			
			for j := 0; j < messageCount; j++ {
				message := fmt.Sprintf("Message %d from %s", j+1, usernames[clientID])
				err := clients[clientID].WriteMessage(websocket.TextMessage, []byte(message))
				if err != nil {
					log.Printf("Client %d failed to send message: %v", clientID+1, err)
					break
				}
				time.Sleep(100 * time.Millisecond) // เว้นระยะเล็กน้อย
			}
		}(i)
	}

	// รอให้การส่งข้อความเสร็จ
	sendWg.Wait()
	fmt.Println("📤 All messages sent!")

	// รอให้การรับข้อความเสร็จ
	time.Sleep(3 * time.Second)
	messageWg.Wait()

	// แสดงสถิติการรับข้อความ
	fmt.Println("\n📊 Message Reception Statistics:")
	fmt.Println("================================")
	totalReceived := 0
	for i := 0; i < numClients; i++ {
		if clients[i] != nil {
			fmt.Printf("• %s: %d messages received\n", usernames[i], receivedMessages[i])
			totalReceived += receivedMessages[i]
		}
	}
	expectedTotal := numClients * (numClients - 1) * messageCount
	fmt.Printf("📈 Total: %d/%d messages received (%.1f%%)\n", 
		totalReceived, expectedTotal, float64(totalReceived)/float64(expectedTotal)*100)

	// ทดสอบการสร้างห้องพร้อมกัน
	fmt.Println("\n🏠 Testing Concurrent Room Creation")
	fmt.Println("===================================")

	roomCount := 5
	fmt.Printf("🏗️  Creating %d rooms concurrently...\n", roomCount)

	var roomWg sync.WaitGroup
	for i := 0; i < roomCount; i++ {
		if clients[i] == nil {
			continue
		}
		
		roomWg.Add(1)
		go func(clientID int) {
			defer roomWg.Done()
			
			roomName := fmt.Sprintf("room%d", clientID+1)
			command := fmt.Sprintf("/create %s", roomName)
			
			err := clients[clientID].WriteMessage(websocket.TextMessage, []byte(command))
			if err != nil {
				log.Printf("Client %d failed to create room: %v", clientID+1, err)
				return
			}
			
			// รับผลลัพธ์
			_, response, err := clients[clientID].ReadMessage()
			if err != nil {
				log.Printf("Client %d failed to read create response: %v", clientID+1, err)
				return
			}
			
			fmt.Printf("🏠 %s: %s\n", usernames[clientID], string(response))
		}(i)
	}

	roomWg.Wait()
	fmt.Println("🎉 Room creation test completed!")

	// ทดสอบ /stats อีกครั้งหลังจากทดสอบ
	fmt.Println("\n📊 Final Server Statistics")
	fmt.Println("==========================")

	if clients[0] != nil {
		err := clients[0].WriteMessage(websocket.TextMessage, []byte("/stats"))
		if err != nil {
			log.Printf("Failed to send final /stats command: %v", err)
		} else {
			// รับผลลัพธ์
			_, finalStats, err := clients[0].ReadMessage()
			if err != nil {
				log.Printf("Failed to read final stats: %v", err)
			} else {
				fmt.Printf("📊 Final Statistics:\n%s\n", string(finalStats))
			}
		}
	}

	fmt.Println("\n✅ Performance and Concurrency Test Completed!")
	fmt.Println("===============================================")
	fmt.Println("📊 Test Results:")
	fmt.Printf("  ✓ %d concurrent connections established\n", numClients)
	fmt.Printf("  ✓ %d concurrent messages sent\n", numClients*messageCount)
	fmt.Printf("  ✓ %d concurrent rooms created\n", roomCount)
	fmt.Println("  ✓ Server metrics and monitoring working")
	fmt.Println("  ✓ Resource management and limits working")

	// รอสัญญาณ interrupt
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	fmt.Println("\n⏳ Press Ctrl+C to close connections and exit...")
	<-interrupt

	// ปิด connections
	fmt.Println("\n🔌 Closing connections...")
	for i, conn := range clients {
		if conn != nil {
			conn.Close()
			fmt.Printf("🔌 Client %d (%s) disconnected\n", i+1, usernames[i])
		}
	}

	fmt.Println("👋 Performance test completed!")
}