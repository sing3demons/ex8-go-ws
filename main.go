package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"realtime-chat/internal/chat"
	"realtime-chat/internal/config"
	wsocket "realtime-chat/internal/websocket"

	"github.com/gorilla/websocket"
)

// roomServiceAdapter adapts chat.RoomService to websocket.RoomService
type roomServiceAdapter struct {
	chatRoomService chat.RoomService
}

func (r *roomServiceAdapter) LeaveRoom(user interface{}, roomName string) error {
	if chatUser, ok := user.(*chat.User); ok {
		return r.chatRoomService.LeaveRoom(chatUser, roomName)
	}
	return nil
}

// wsManagerAdapter adapts websocket.Manager to chat.WebSocketManager
type wsManagerAdapter struct {
	wsManager *wsocket.Manager
}

func (w *wsManagerAdapter) AddConnection(conn *websocket.Conn) string {
	return w.wsManager.AddConnection(conn)
}

func (w *wsManagerAdapter) RemoveConnection(connID string) {
	w.wsManager.RemoveConnection(connID)
}

func (w *wsManagerAdapter) GetConnection(connID string) (chat.Connection, bool) {
	conn, exists := w.wsManager.GetConnection(connID)
	return conn, exists
}

func (w *wsManagerAdapter) BroadcastMessage(message interface{}, excludeID string) {
	w.wsManager.BroadcastMessage(message, excludeID)
}

func (w *wsManagerAdapter) BroadcastToRoom(message interface{}, excludeID, roomName string) {
	w.wsManager.BroadcastToRoom(message, excludeID, roomName)
}

func (w *wsManagerAdapter) GetConnectionHealth(connID string) (*config.ConnectionHealth, bool) {
	return w.wsManager.GetConnectionHealth(connID)
}

func main() {
	// สร้าง configuration manager
	configManager := config.NewConfigManager("config.json")

	// โหลด configuration
	if err := configManager.Initialize(); err != nil {
		log.Printf("⚠️ Failed to initialize config manager: %v", err)
		log.Println("🔄 Using default configuration")
	}

	// ดึง configuration
	cfg := configManager.GetConfig()

	// สร้าง metrics
	metrics := config.NewServerMetrics()

	// สร้าง repositories
	userRepo := chat.NewInMemoryUserRepository()
	roomRepo := chat.NewInMemoryRoomRepository()

	// สร้าง services
	userService := chat.NewUserService(userRepo, metrics)
	roomService := chat.NewRoomService(roomRepo, cfg.MaxRooms, cfg.MaxUsersPerRoom, metrics)

	// สร้าง WebSocket manager
	wsManager := wsocket.NewManager(cfg, userService, &roomServiceAdapter{roomService}, metrics)

	// สร้าง adapter สำหรับ WebSocket manager
	wsManagerAdapted := &wsManagerAdapter{wsManager}

	// สร้าง message service
	messageService := chat.NewMessageService(wsManagerAdapted)

	// สร้าง command service
	commandService := chat.NewCommandService(userService, roomService, messageService, wsManagerAdapted, metrics, cfg, configManager)

	// สร้าง HTTP handler
	handler := chat.NewHandler(wsManagerAdapted, userService, roomService, commandService, messageService, cfg)

	// เริ่ม WebSocket manager ใน goroutine
	go wsManager.Run()

	// เริ่ม metrics logging goroutine
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.Printf("📊 Active connections: %d", wsManager.GetConnectionCount())
			}
		}
	}()

	// ตั้งค่า HTTP routes
	http.HandleFunc("/ws", handler.HandleWebSocket)

	// เสิร์ฟ static files สำหรับ test client
	http.Handle("/", http.FileServer(http.Dir("./static/")))

	// สร้าง HTTP server
	port := cfg.Port
	if port[0] != ':' {
		port = ":" + port
	}
	server := &http.Server{
		Addr:         port,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	// ตั้งค่า graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		sig := <-sigChan
		log.Printf("🛑 Received signal: %v", sig)
		log.Println("🔄 Starting graceful shutdown...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("❌ Server shutdown error: %v", err)
		} else {
			log.Println("✅ Server shutdown completed")
		}
	}()

	log.Printf("🚀 Starting WebSocket Chat Server on port %s", cfg.Port)
	log.Printf("📡 WebSocket endpoint: ws://localhost%s/ws", cfg.Port)
	log.Printf("🌐 Test page: http://localhost%s", cfg.Port)
	log.Printf("👥 Connection Manager: Ready (Max: %d)", cfg.MaxConnections)
	log.Printf("🔐 User Manager: Ready")
	log.Printf("🏠 Room Manager: Ready (Max: %d)", cfg.MaxRooms)
	log.Printf("📋 Command Handler: Ready")
	log.Printf("📊 Message Service: Ready")
	log.Printf("⚙️  Configuration: Heartbeat=%v, ReadTimeout=%v, WriteTimeout=%v",
		cfg.HeartbeatInterval, cfg.ReadTimeout, cfg.WriteTimeout)

	log.Println("🛑 Press Ctrl+C for graceful shutdown")

	// เริ่ม server
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("❌ Server failed to start: %v", err)
	}

	log.Println("👋 Server stopped gracefully")
}
