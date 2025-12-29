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
	"realtime-chat/internal/database"
	"realtime-chat/internal/message"
	"realtime-chat/internal/room"
	"realtime-chat/internal/user"
	userPkg "realtime-chat/internal/user"
	wsocket "realtime-chat/internal/websocket"

	"github.com/gorilla/websocket"
)

// wsRoomServiceAdapter adapts room.Service to websocket.RoomService
type wsRoomServiceAdapter struct {
	roomService room.Service
}

func (r *wsRoomServiceAdapter) LeaveRoom(user interface{}, roomName string) error {
	if chatUser, ok := user.(*userPkg.User); ok {
		return r.roomService.LeaveRoom(chatUser, roomName)
	}
	return nil
}

// wsManagerAdapter adapts websocket.Manager to chat.WebSocketManager
type wsManagerAdapter struct {
	wsManager *wsocket.Manager
}

func (w *wsManagerAdapter) AddConnection(conn interface{}) string {
	if wsConn, ok := conn.(*websocket.Conn); ok {
		return w.wsManager.AddConnection(wsConn)
	}
	return ""
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

func (w *wsManagerAdapter) GetConnectionHealth(connID string) (interface{}, bool) {
	health, exists := w.wsManager.GetConnectionHealth(connID)
	return health, exists
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
	var userRepo user.Repository
	var roomRepo room.Repository
	var messageRepo message.Repository
	var mongoDB *database.MongoDB

	if cfg.EnableMongoDB {
		log.Println("🔄 Initializing MongoDB connection...")

		// สร้าง MongoDB configuration
		mongoConfig := &database.MongoConfig{
			URI:            cfg.MongoURI,
			Database:       cfg.MongoDatabase,
			ConnectTimeout: cfg.MongoConnectTimeout,
			PingTimeout:    cfg.MongoPingTimeout,
			MaxPoolSize:    cfg.MongoMaxPoolSize,
			MinPoolSize:    cfg.MongoMinPoolSize,
		}

		// เชื่อมต่อ MongoDB
		var err error
		mongoDB, err = database.NewMongoDB(mongoConfig)
		if err != nil {
			log.Printf("❌ Failed to connect to MongoDB: %v", err)
			log.Println("🔄 Falling back to in-memory repositories")
			cfg.EnableMongoDB = false
		} else {
			// สร้าง indexes
			if err := mongoDB.CreateIndexes(); err != nil {
				log.Printf("⚠️ Failed to create MongoDB indexes: %v", err)
			}

			// สร้าง MongoDB repositories
			userRepo = user.NewMongoRepository(mongoDB)
			roomRepo = room.NewMongoRepository(mongoDB)
			messageRepo = message.NewMongoRepository(mongoDB)

			log.Println("✅ MongoDB repositories initialized")
		}
	}

	// ถ้าไม่ใช้ MongoDB หรือเชื่อมต่อไม่ได้ ให้ใช้ in-memory repositories
	if !cfg.EnableMongoDB {
		log.Println("🔄 Using in-memory repositories")
		userRepo = user.NewInMemoryRepository()
		roomRepo = room.NewInMemoryRepository()
	}

	// สร้าง services
	userService := user.NewService(userRepo, metrics)
	roomService := room.NewService(roomRepo, cfg.MaxRooms, cfg.MaxUsersPerRoom, metrics)

	// สร้าง WebSocket manager
	wsRoomAdapter := &wsRoomServiceAdapter{roomService}
	wsManager := wsocket.NewManager(cfg, userService, wsRoomAdapter, metrics)

	// สร้าง adapter สำหรับ WebSocket manager
	wsManagerAdapted := &wsManagerAdapter{wsManager}

	// สร้าง message service
	messageService := chat.NewMessageService(wsManagerAdapted)

	// สร้าง command service
	commandService := chat.NewCommandService(userService, roomService, messageService, wsManagerAdapted, metrics, cfg, configManager)

	// สร้าง HTTP handler
	handler := chat.NewHandler(wsManagerAdapted, userService, roomService, commandService, messageService, cfg)

	// Set message repository if MongoDB is enabled
	if cfg.EnableMongoDB && messageRepo != nil {
		commandService.SetMessageRepository(messageRepo)
		handler.SetMessageRepository(messageRepo)
		log.Println("✅ Message persistence enabled")
	}

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

		// ปิด MongoDB connection ถ้ามี
		if mongoDB != nil {
			if err := mongoDB.Close(); err != nil {
				log.Printf("⚠️ Error closing MongoDB connection: %v", err)
			}
		}

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

	if cfg.EnableMongoDB && mongoDB != nil {
		log.Printf("🗄️  Database: MongoDB (%s/%s)", cfg.MongoURI, cfg.MongoDatabase)
	} else {
		log.Printf("🗄️  Database: In-Memory")
	}

	log.Printf("⚙️  Configuration: Heartbeat=%v, ReadTimeout=%v, WriteTimeout=%v",
		cfg.HeartbeatInterval, cfg.ReadTimeout, cfg.WriteTimeout)

	log.Println("🛑 Press Ctrl+C for graceful shutdown")

	// เริ่ม server
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("❌ Server failed to start: %v", err)
	}

	log.Println("👋 Server stopped gracefully")
}
