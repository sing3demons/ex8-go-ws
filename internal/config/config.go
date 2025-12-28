package config

import (
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