package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	fmt.Println("🧪 Testing User Management System...")
	fmt.Println("👥 Creating multiple users with different scenarios...")

	var wg sync.WaitGroup

	// Test Case 1: Normal user registration
	wg.Add(1)
	go func() {
		defer wg.Done()
		testNormalUser("Alice", 1)
	}()

	// Test Case 2: Another normal user
	wg.Add(1)
	go func() {
		defer wg.Done()
		testNormalUser("Bob", 2)
	}()

	// Test Case 3: Duplicate username (should fail)
	time.Sleep(2 * time.Second) // รอให้ Alice ลงทะเบียนก่อน
	wg.Add(1)
	go func() {
		defer wg.Done()
		testDuplicateUser("Alice", 3)
	}()

	// Test Case 4: Empty username (should fail)
	wg.Add(1)
	go func() {
		defer wg.Done()
		testEmptyUser(4)
	}()

	// รอให้ users ลงทะเบียนเสร็จ
	time.Sleep(3 * time.Second)

	// Test messaging between users
	fmt.Println("\n💬 Testing messaging between authenticated users...")
	
	wg.Wait()
	fmt.Println("\n✅ User Management System test completed!")
}

func testNormalUser(username string, clientID int) {
	serverURL := "ws://localhost:9090/ws"
	
	conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		log.Printf("❌ Client %d failed to connect: %v", clientID, err)
		return
	}
	defer conn.Close()

	fmt.Printf("✅ Client %d connected\n", clientID)

	// Goroutine สำหรับรับข้อความ
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			fmt.Printf("📨 Client %d (%s) received: %s\n", clientID, username, string(message))
		}
	}()

	// รอรับข้อความขอ username
	time.Sleep(500 * time.Millisecond)

	// ส่ง username
	err = conn.WriteMessage(websocket.TextMessage, []byte(username))
	if err != nil {
		log.Printf("❌ Client %d failed to send username: %v", clientID, err)
		return
	}
	fmt.Printf("📤 Client %d sent username: %s\n", clientID, username)

	// รอการ authentication
	time.Sleep(1 * time.Second)

	// ส่งข้อความทดสอบ
	testMessage := fmt.Sprintf("Hello everyone! This is %s 👋", username)
	err = conn.WriteMessage(websocket.TextMessage, []byte(testMessage))
	if err != nil {
		log.Printf("❌ Client %d failed to send message: %v", clientID, err)
		return
	}
	fmt.Printf("📤 Client %d (%s) sent: %s\n", clientID, username, testMessage)

	// รอรับข้อความจาก users อื่น
	time.Sleep(5 * time.Second)
}

func testDuplicateUser(username string, clientID int) {
	serverURL := "ws://localhost:9090/ws"
	
	conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		log.Printf("❌ Client %d failed to connect: %v", clientID, err)
		return
	}
	defer conn.Close()

	fmt.Printf("✅ Client %d connected (testing duplicate username)\n", clientID)

	// Goroutine สำหรับรับข้อความ
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			fmt.Printf("📨 Client %d (duplicate test) received: %s\n", clientID, string(message))
		}
	}()

	// รอรับข้อความขอ username
	time.Sleep(500 * time.Millisecond)

	// ส่ง username ที่ซ้ำ
	err = conn.WriteMessage(websocket.TextMessage, []byte(username))
	if err != nil {
		log.Printf("❌ Client %d failed to send username: %v", clientID, err)
		return
	}
	fmt.Printf("📤 Client %d sent duplicate username: %s (should be rejected)\n", clientID, username)

	// รอการตอบกลับ
	time.Sleep(2 * time.Second)

	// ลองส่ง username ใหม่
	newUsername := username + "2"
	err = conn.WriteMessage(websocket.TextMessage, []byte(newUsername))
	if err != nil {
		log.Printf("❌ Client %d failed to send new username: %v", clientID, err)
		return
	}
	fmt.Printf("📤 Client %d sent new username: %s\n", clientID, newUsername)

	time.Sleep(3 * time.Second)
}

func testEmptyUser(clientID int) {
	serverURL := "ws://localhost:9090/ws"
	
	conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		log.Printf("❌ Client %d failed to connect: %v", clientID, err)
		return
	}
	defer conn.Close()

	fmt.Printf("✅ Client %d connected (testing empty username)\n", clientID)

	// Goroutine สำหรับรับข้อความ
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			fmt.Printf("📨 Client %d (empty test) received: %s\n", clientID, string(message))
		}
	}()

	// รอรับข้อความขอ username
	time.Sleep(500 * time.Millisecond)

	// ส่ง username ว่าง
	err = conn.WriteMessage(websocket.TextMessage, []byte(""))
	if err != nil {
		log.Printf("❌ Client %d failed to send empty username: %v", clientID, err)
		return
	}
	fmt.Printf("📤 Client %d sent empty username (should be rejected)\n", clientID)

	// รอการตอบกลับ
	time.Sleep(1 * time.Second)

	// ส่ง username ที่ถูกต้อง
	validUsername := "Charlie"
	err = conn.WriteMessage(websocket.TextMessage, []byte(validUsername))
	if err != nil {
		log.Printf("❌ Client %d failed to send valid username: %v", clientID, err)
		return
	}
	fmt.Printf("📤 Client %d sent valid username: %s\n", clientID, validUsername)

	time.Sleep(3 * time.Second)
}