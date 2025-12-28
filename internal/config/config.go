package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
)

// ConnectionHealth represents the health status of a connection
type ConnectionHealth struct {
	IsHealthy        bool      `json:"is_healthy"`
	LastPingTime     time.Time `json:"last_ping_time"`
	LastPongTime     time.Time `json:"last_pong_time"`
	PingsSent        int64     `json:"pings_sent"`
	PongsReceived    int64     `json:"pongs_received"`
	MissedPongs      int64     `json:"missed_pongs"`
	ConnectionStart  time.Time `json:"connection_start"`
	LastActivity     time.Time `json:"last_activity"`
	mutex            sync.RWMutex
}

// NewConnectionHealth creates a new connection health tracker
func NewConnectionHealth() *ConnectionHealth {
	now := time.Now()
	return &ConnectionHealth{
		IsHealthy:       true,
		ConnectionStart: now,
		LastActivity:    now,
	}
}

// RecordPing records a ping sent
func (ch *ConnectionHealth) RecordPing() {
	ch.mutex.Lock()
	defer ch.mutex.Unlock()
	ch.LastPingTime = time.Now()
	ch.PingsSent++
}

// RecordPong records a pong received
func (ch *ConnectionHealth) RecordPong() {
	ch.mutex.Lock()
	defer ch.mutex.Unlock()
	ch.LastPongTime = time.Now()
	ch.PongsReceived++
	ch.IsHealthy = true
	ch.MissedPongs = 0
}

// RecordActivity records general activity
func (ch *ConnectionHealth) RecordActivity() {
	ch.mutex.Lock()
	defer ch.mutex.Unlock()
	ch.LastActivity = time.Now()
}

// CheckHealth checks if connection is healthy
func (ch *ConnectionHealth) CheckHealth(pongTimeout time.Duration) bool {
	ch.mutex.Lock()
	defer ch.mutex.Unlock()
	
	// ถ้าไม่เคยส่ง ping ถือว่า healthy
	if ch.LastPingTime.IsZero() {
		return true
	}
	
	// ถ้าไม่เคยได้รับ pong แต่ส่ง ping ไปแล้ว
	if ch.LastPongTime.IsZero() && !ch.LastPingTime.IsZero() {
		// ตรวจสอบว่าเกิน timeout หรือไม่
		if time.Since(ch.LastPingTime) > pongTimeout {
			ch.IsHealthy = false
			ch.MissedPongs++
			return false
		}
		return true
	}
	
	// ตรวจสอบว่า pong ล่าสุดเก่าเกินไปหรือไม่
	if time.Since(ch.LastPongTime) > pongTimeout {
		ch.IsHealthy = false
		ch.MissedPongs++
		return false
	}
	
	return ch.IsHealthy
}

// GetStats returns health statistics
func (ch *ConnectionHealth) GetStats() *ConnectionHealth {
	ch.mutex.RLock()
	defer ch.mutex.RUnlock()
	
	return &ConnectionHealth{
		IsHealthy:       ch.IsHealthy,
		LastPingTime:    ch.LastPingTime,
		LastPongTime:    ch.LastPongTime,
		PingsSent:       ch.PingsSent,
		PongsReceived:   ch.PongsReceived,
		MissedPongs:     ch.MissedPongs,
		ConnectionStart: ch.ConnectionStart,
		LastActivity:    ch.LastActivity,
	}
}

// ServerConfig holds server configuration
type ServerConfig struct {
	MaxConnections      int           `json:"max_connections"`
	MaxRooms            int           `json:"max_rooms"`
	MaxUsersPerRoom     int           `json:"max_users_per_room"`
	HeartbeatInterval   time.Duration `json:"heartbeat_interval"`
	ReadTimeout         time.Duration `json:"read_timeout"`
	WriteTimeout        time.Duration `json:"write_timeout"`
	PongTimeout         time.Duration `json:"pong_timeout"`
	ConnectionTimeout   time.Duration `json:"connection_timeout"`
	BroadcastBuffer     int           `json:"broadcast_buffer"`
	EnableMetrics       bool          `json:"enable_metrics"`
	EnableHealthCheck   bool          `json:"enable_health_check"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	Port                string        `json:"port"`
	
	// Security settings
	MaxMessageLength    int           `json:"max_message_length"`
	MaxUsernameLength   int           `json:"max_username_length"`
	MaxRoomNameLength   int           `json:"max_room_name_length"`
	RateLimitMessages   int           `json:"rate_limit_messages"`
	RateLimitWindow     time.Duration `json:"rate_limit_window"`
	EnableRateLimit     bool          `json:"enable_rate_limit"`
}

// DefaultServerConfig returns default server configuration
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		MaxConnections:      1000,
		MaxRooms:            100,
		MaxUsersPerRoom:     50,
		HeartbeatInterval:   30 * time.Second,  // ลดลงเพื่อตรวจสอบบ่อยขึ้น
		ReadTimeout:         60 * time.Second,
		WriteTimeout:        10 * time.Second,
		PongTimeout:         60 * time.Second,  // เวลารอ pong response
		ConnectionTimeout:   5 * time.Minute,  // timeout สำหรับ inactive connections
		BroadcastBuffer:     256,
		EnableMetrics:       true,
		EnableHealthCheck:   true,
		HealthCheckInterval: 30 * time.Second,  // ตรวจสอบ health ทุก 30 วินาที
		Port:                ":9090",
		
		// Security settings
		MaxMessageLength:    1000,              // จำกัดความยาวข้อความ
		MaxUsernameLength:   50,                // จำกัดความยาว username
		MaxRoomNameLength:   50,                // จำกัดความยาวชื่อห้อง
		RateLimitMessages:   10,                // จำกัด 10 ข้อความ
		RateLimitWindow:     1 * time.Minute,   // ต่อ 1 นาที
		EnableRateLimit:     true,              // เปิดใช้ rate limiting
	}
}

// ServerMetrics holds server performance metrics
type ServerMetrics struct {
	TotalConnections    int64     `json:"total_connections"`
	ActiveConnections   int64     `json:"active_connections"`
	TotalMessages       int64     `json:"total_messages"`
	TotalCommands       int64     `json:"total_commands"`
	TotalRooms          int64     `json:"total_rooms"`
	TotalUsers          int64     `json:"total_users"`
	StartTime           time.Time `json:"start_time"`
	LastMessageTime     time.Time `json:"last_message_time"`
	MessageRate         float64   `json:"message_rate"`
	ConnectionRate      float64   `json:"connection_rate"`
	mutex               sync.RWMutex
}

// NewServerMetrics creates new server metrics
func NewServerMetrics() *ServerMetrics {
	return &ServerMetrics{
		StartTime: time.Now(),
	}
}

// IncrementConnections increments connection count
func (sm *ServerMetrics) IncrementConnections() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.TotalConnections++
	sm.ActiveConnections++
}

// DecrementConnections decrements active connection count
func (sm *ServerMetrics) DecrementConnections() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.ActiveConnections--
}

// IncrementMessages increments message count
func (sm *ServerMetrics) IncrementMessages() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.TotalMessages++
	sm.LastMessageTime = time.Now()
}

// IncrementCommands increments command count
func (sm *ServerMetrics) IncrementCommands() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.TotalCommands++
}

// IncrementRooms increments room count
func (sm *ServerMetrics) IncrementRooms() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.TotalRooms++
}

// IncrementUsers increments user count
func (sm *ServerMetrics) IncrementUsers() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.TotalUsers++
}

// DecrementUsers decrements user count
func (sm *ServerMetrics) DecrementUsers() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.TotalUsers--
}

// GetMetrics returns current metrics with calculated rates
func (sm *ServerMetrics) GetMetrics() *ServerMetrics {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	
	// Calculate rates
	uptime := time.Since(sm.StartTime).Seconds()
	messageRate := float64(sm.TotalMessages) / uptime
	connectionRate := float64(sm.TotalConnections) / uptime
	
	return &ServerMetrics{
		TotalConnections:  sm.TotalConnections,
		ActiveConnections: sm.ActiveConnections,
		TotalMessages:     sm.TotalMessages,
		TotalCommands:     sm.TotalCommands,
		TotalRooms:        sm.TotalRooms,
		TotalUsers:        sm.TotalUsers,
		StartTime:         sm.StartTime,
		LastMessageTime:   sm.LastMessageTime,
		MessageRate:       messageRate,
		ConnectionRate:    connectionRate,
	}
}

// RateLimiter manages rate limiting per user
type RateLimiter struct {
	limits map[string]*UserRateLimit
	mutex  sync.RWMutex
	config *ServerConfig
}

// UserRateLimit tracks rate limiting for a specific user
type UserRateLimit struct {
	MessageCount int
	WindowStart  time.Time
	mutex        sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *ServerConfig) *RateLimiter {
	return &RateLimiter{
		limits: make(map[string]*UserRateLimit),
		config: config,
	}
}

// CheckRateLimit checks if a user can send a message
func (rl *RateLimiter) CheckRateLimit(userID string) bool {
	if !rl.config.EnableRateLimit {
		return true
	}

	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	
	// Get or create user rate limit
	userLimit, exists := rl.limits[userID]
	if !exists {
		userLimit = &UserRateLimit{
			MessageCount: 0,
			WindowStart:  now,
		}
		rl.limits[userID] = userLimit
	}

	userLimit.mutex.Lock()
	defer userLimit.mutex.Unlock()

	// Check if window has expired
	if now.Sub(userLimit.WindowStart) > rl.config.RateLimitWindow {
		// Reset window
		userLimit.MessageCount = 0
		userLimit.WindowStart = now
	}

	// Check if user has exceeded limit
	if userLimit.MessageCount >= rl.config.RateLimitMessages {
		return false
	}

	// Increment message count
	userLimit.MessageCount++
	return true
}

// GetRateLimitStatus returns current rate limit status for a user
func (rl *RateLimiter) GetRateLimitStatus(userID string) (int, int, time.Duration) {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	userLimit, exists := rl.limits[userID]
	if !exists {
		return 0, rl.config.RateLimitMessages, rl.config.RateLimitWindow
	}

	userLimit.mutex.Lock()
	defer userLimit.mutex.Unlock()

	remaining := rl.config.RateLimitMessages - userLimit.MessageCount
	if remaining < 0 {
		remaining = 0
	}

	timeRemaining := rl.config.RateLimitWindow - time.Since(userLimit.WindowStart)
	if timeRemaining < 0 {
		timeRemaining = 0
	}

	return remaining, rl.config.RateLimitMessages, timeRemaining
}

// ConfigLoader handles loading configuration from various sources
type ConfigLoader struct {
	configPath string
	mutex      sync.RWMutex
}

// NewConfigLoader creates a new configuration loader
func NewConfigLoader(configPath string) *ConfigLoader {
	return &ConfigLoader{
		configPath: configPath,
	}
}

// LoadConfig loads configuration from file and environment variables
func (cl *ConfigLoader) LoadConfig() (*ServerConfig, error) {
	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	// Start with default configuration
	config := DefaultServerConfig()

	// Try to load from file if it exists
	if cl.configPath != "" {
		if err := cl.loadFromFile(config); err != nil {
			// Log error but continue with defaults
			fmt.Printf("⚠️ Failed to load config file %s: %v\n", cl.configPath, err)
		}
	}

	// Override with environment variables
	cl.loadFromEnv(config)

	return config, nil
}

// loadFromFile loads configuration from JSON file
func (cl *ConfigLoader) loadFromFile(config *ServerConfig) error {
	if _, err := os.Stat(cl.configPath); os.IsNotExist(err) {
		return fmt.Errorf("config file does not exist: %s", cl.configPath)
	}

	data, err := os.ReadFile(cl.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	fmt.Printf("✅ Loaded configuration from %s\n", cl.configPath)
	return nil
}

// loadFromEnv loads configuration from environment variables
func (cl *ConfigLoader) loadFromEnv(config *ServerConfig) {
	// Server settings
	if port := os.Getenv("CHAT_PORT"); port != "" {
		config.Port = port
	}
	
	if maxConn := os.Getenv("CHAT_MAX_CONNECTIONS"); maxConn != "" {
		if val, err := strconv.Atoi(maxConn); err == nil {
			config.MaxConnections = val
		}
	}
	
	if maxRooms := os.Getenv("CHAT_MAX_ROOMS"); maxRooms != "" {
		if val, err := strconv.Atoi(maxRooms); err == nil {
			config.MaxRooms = val
		}
	}
	
	if maxUsers := os.Getenv("CHAT_MAX_USERS_PER_ROOM"); maxUsers != "" {
		if val, err := strconv.Atoi(maxUsers); err == nil {
			config.MaxUsersPerRoom = val
		}
	}

	// Timeout settings
	if heartbeat := os.Getenv("CHAT_HEARTBEAT_INTERVAL"); heartbeat != "" {
		if val, err := time.ParseDuration(heartbeat); err == nil {
			config.HeartbeatInterval = val
		}
	}
	
	if readTimeout := os.Getenv("CHAT_READ_TIMEOUT"); readTimeout != "" {
		if val, err := time.ParseDuration(readTimeout); err == nil {
			config.ReadTimeout = val
		}
	}
	
	if writeTimeout := os.Getenv("CHAT_WRITE_TIMEOUT"); writeTimeout != "" {
		if val, err := time.ParseDuration(writeTimeout); err == nil {
			config.WriteTimeout = val
		}
	}

	// Security settings
	if maxMsgLen := os.Getenv("CHAT_MAX_MESSAGE_LENGTH"); maxMsgLen != "" {
		if val, err := strconv.Atoi(maxMsgLen); err == nil {
			config.MaxMessageLength = val
		}
	}
	
	if maxUserLen := os.Getenv("CHAT_MAX_USERNAME_LENGTH"); maxUserLen != "" {
		if val, err := strconv.Atoi(maxUserLen); err == nil {
			config.MaxUsernameLength = val
		}
	}
	
	if rateLimitMsg := os.Getenv("CHAT_RATE_LIMIT_MESSAGES"); rateLimitMsg != "" {
		if val, err := strconv.Atoi(rateLimitMsg); err == nil {
			config.RateLimitMessages = val
		}
	}
	
	if rateLimitWindow := os.Getenv("CHAT_RATE_LIMIT_WINDOW"); rateLimitWindow != "" {
		if val, err := time.ParseDuration(rateLimitWindow); err == nil {
			config.RateLimitWindow = val
		}
	}

	// Feature flags
	if enableMetrics := os.Getenv("CHAT_ENABLE_METRICS"); enableMetrics != "" {
		config.EnableMetrics = enableMetrics == "true"
	}
	
	if enableHealth := os.Getenv("CHAT_ENABLE_HEALTH_CHECK"); enableHealth != "" {
		config.EnableHealthCheck = enableHealth == "true"
	}
	
	if enableRateLimit := os.Getenv("CHAT_ENABLE_RATE_LIMIT"); enableRateLimit != "" {
		config.EnableRateLimit = enableRateLimit == "true"
	}
}

// SaveConfig saves current configuration to file
func (cl *ConfigLoader) SaveConfig(config *ServerConfig) error {
	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(cl.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	fmt.Printf("✅ Saved configuration to %s\n", cl.configPath)
	return nil
}

// WatchConfig watches for configuration file changes (basic implementation)
func (cl *ConfigLoader) WatchConfig(callback func(*ServerConfig)) error {
	// This is a basic implementation - in production you might use fsnotify
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		
		var lastModTime time.Time
		if stat, err := os.Stat(cl.configPath); err == nil {
			lastModTime = stat.ModTime()
		}
		
		for range ticker.C {
			if stat, err := os.Stat(cl.configPath); err == nil {
				if stat.ModTime().After(lastModTime) {
					lastModTime = stat.ModTime()
					if newConfig, err := cl.LoadConfig(); err == nil {
						fmt.Println("🔄 Configuration file changed, reloading...")
						callback(newConfig)
					}
				}
			}
		}
	}()
	
	return nil
}