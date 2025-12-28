package config

import (
	"fmt"
	"sync"
	"time"
)

// ConfigManager manages runtime configuration updates
type ConfigManager struct {
	config    *ServerConfig
	loader    *ConfigLoader
	mutex     sync.RWMutex
	callbacks []func(*ServerConfig)
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(configPath string) *ConfigManager {
	return &ConfigManager{
		loader:    NewConfigLoader(configPath),
		callbacks: make([]func(*ServerConfig), 0),
	}
}

// Initialize loads initial configuration
func (cm *ConfigManager) Initialize() error {
	config, err := cm.loader.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to initialize config: %v", err)
	}

	cm.mutex.Lock()
	cm.config = config
	cm.mutex.Unlock()

	// Start watching for changes
	return cm.loader.WatchConfig(cm.onConfigChange)
}

// GetConfig returns current configuration (thread-safe)
func (cm *ConfigManager) GetConfig() *ServerConfig {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	
	// Return a copy to prevent external modifications
	configCopy := *cm.config
	return &configCopy
}

// UpdateConfig updates configuration at runtime
func (cm *ConfigManager) UpdateConfig(updates map[string]interface{}) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Apply updates to current config
	if err := cm.applyUpdates(updates); err != nil {
		return fmt.Errorf("failed to apply config updates: %v", err)
	}

	// Save to file
	if err := cm.loader.SaveConfig(cm.config); err != nil {
		return fmt.Errorf("failed to save config: %v", err)
	}

	// Notify callbacks
	cm.notifyCallbacks()

	return nil
}

// RegisterCallback registers a callback for configuration changes
func (cm *ConfigManager) RegisterCallback(callback func(*ServerConfig)) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.callbacks = append(cm.callbacks, callback)
}

// onConfigChange handles configuration file changes
func (cm *ConfigManager) onConfigChange(newConfig *ServerConfig) {
	cm.mutex.Lock()
	cm.config = newConfig
	cm.mutex.Unlock()

	cm.notifyCallbacks()
}

// notifyCallbacks notifies all registered callbacks
func (cm *ConfigManager) notifyCallbacks() {
	configCopy := *cm.config
	for _, callback := range cm.callbacks {
		go callback(&configCopy)
	}
}

// applyUpdates applies configuration updates
func (cm *ConfigManager) applyUpdates(updates map[string]interface{}) error {
	for key, value := range updates {
		switch key {
		case "max_connections":
			if val, ok := value.(float64); ok {
				cm.config.MaxConnections = int(val)
			}
		case "max_rooms":
			if val, ok := value.(float64); ok {
				cm.config.MaxRooms = int(val)
			}
		case "max_users_per_room":
			if val, ok := value.(float64); ok {
				cm.config.MaxUsersPerRoom = int(val)
			}
		case "heartbeat_interval":
			if val, ok := value.(string); ok {
				if duration, err := time.ParseDuration(val); err == nil {
					cm.config.HeartbeatInterval = duration
				}
			}
		case "max_message_length":
			if val, ok := value.(float64); ok {
				cm.config.MaxMessageLength = int(val)
			}
		case "max_username_length":
			if val, ok := value.(float64); ok {
				cm.config.MaxUsernameLength = int(val)
			}
		case "rate_limit_messages":
			if val, ok := value.(float64); ok {
				cm.config.RateLimitMessages = int(val)
			}
		case "rate_limit_window":
			if val, ok := value.(string); ok {
				if duration, err := time.ParseDuration(val); err == nil {
					cm.config.RateLimitWindow = duration
				}
			}
		case "enable_metrics":
			if val, ok := value.(bool); ok {
				cm.config.EnableMetrics = val
			}
		case "enable_health_check":
			if val, ok := value.(bool); ok {
				cm.config.EnableHealthCheck = val
			}
		case "enable_rate_limit":
			if val, ok := value.(bool); ok {
				cm.config.EnableRateLimit = val
			}
		default:
			return fmt.Errorf("unknown configuration key: %s", key)
		}
	}
	return nil
}

// GetConfigSummary returns a summary of current configuration
func (cm *ConfigManager) GetConfigSummary() map[string]interface{} {
	config := cm.GetConfig()
	
	return map[string]interface{}{
		"server": map[string]interface{}{
			"port":                config.Port,
			"max_connections":     config.MaxConnections,
			"max_rooms":          config.MaxRooms,
			"max_users_per_room": config.MaxUsersPerRoom,
		},
		"timeouts": map[string]interface{}{
			"heartbeat_interval": config.HeartbeatInterval.String(),
			"read_timeout":       config.ReadTimeout.String(),
			"write_timeout":      config.WriteTimeout.String(),
			"pong_timeout":       config.PongTimeout.String(),
		},
		"security": map[string]interface{}{
			"max_message_length":  config.MaxMessageLength,
			"max_username_length": config.MaxUsernameLength,
			"max_room_name_length": config.MaxRoomNameLength,
			"rate_limit_messages": config.RateLimitMessages,
			"rate_limit_window":   config.RateLimitWindow.String(),
		},
		"features": map[string]interface{}{
			"enable_metrics":      config.EnableMetrics,
			"enable_health_check": config.EnableHealthCheck,
			"enable_rate_limit":   config.EnableRateLimit,
		},
	}
}