package config

import (
	"os"
	"testing"
	"time"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader()
	if loader == nil {
		t.Fatal("NewLoader returned nil")
	}
	if loader.viper == nil {
		t.Error("viper instance should not be nil")
	}
}

func TestLoader_Load_Defaults(t *testing.T) {
	loader := NewLoader()

	// Load without config file (should use defaults)
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg == nil {
		t.Fatal("Config should not be nil")
	}

	// Verify defaults
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Expected default host '0.0.0.0', got '%s'", cfg.Server.Host)
	}
	if cfg.Engine.QueueSize != 2000 {
		t.Errorf("Expected default queue size 2000, got %d", cfg.Engine.QueueSize)
	}
	if cfg.Cache.L1Size != 5000 {
		t.Errorf("Expected default L1 size 5000, got %d", cfg.Cache.L1Size)
	}
}

func TestGetEngineWorkers(t *testing.T) {
	tests := []struct {
		name     string
		workers  interface{}
		wantAuto bool
		wantNum  int
	}{
		{
			name:     "auto string",
			workers:  "auto",
			wantAuto: true,
			wantNum:  0,
		},
		{
			name:     "explicit number",
			workers:  4,
			wantAuto: false,
			wantNum:  4,
		},
		{
			name:     "nil (auto)",
			workers:  nil,
			wantAuto: true,
			wantNum:  0,
		},
		{
			name:     "zero (auto)",
			workers:  0,
			wantAuto: true,
			wantNum:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &EngineConfig{
				Workers: tt.workers,
			}

			result := GetEngineWorkers(cfg)
			if tt.wantAuto {
				if result != 0 {
					t.Errorf("Expected auto (0), got %d", result)
				}
			} else {
				if result != tt.wantNum {
					t.Errorf("Expected %d workers, got %d", tt.wantNum, result)
				}
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				Server: ServerConfig{
					Port:         8080,
					Host:         "0.0.0.0",
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
				},
				Engine: EngineConfig{
					Workers:   "auto",
					QueueSize: 2000,
					Timeout:   20 * time.Second,
				},
				Cache: CacheConfig{
					L1Size: 5000,
					L2TTL:  time.Hour,
				},
				NATS: NATSConfig{
					URLs: []string{"nats://localhost:4222"},
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid port",
			cfg: &Config{
				Server: ServerConfig{
					Port: 70000, // Invalid port
				},
			},
			wantErr: true,
		},
		{
			name: "invalid queue size",
			cfg: &Config{
				Server: ServerConfig{
					Port: 8080,
				},
				Engine: EngineConfig{
					QueueSize: -1, // Invalid
				},
			},
			wantErr: true,
		},
		{
			name: "empty NATS URLs",
			cfg: &Config{
				Server: ServerConfig{
					Port: 8080,
				},
				Engine: EngineConfig{
					QueueSize: 2000,
				},
				NATS: NATSConfig{
					URLs: []string{}, // Empty
				},
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			cfg: &Config{
				Server: ServerConfig{
					Port: 8080,
				},
				Engine: EngineConfig{
					QueueSize: 2000,
				},
				NATS: NATSConfig{
					URLs: []string{"nats://localhost:4222"},
				},
				Logging: LoggingConfig{
					Level: "invalid", // Invalid level
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoader_Load_EnvironmentVariables(t *testing.T) {
	// Set environment variable with new HULK_ prefix
	os.Setenv("HULK_SERVER_PORT", "9090")
	defer os.Unsetenv("HULK_SERVER_PORT")

	loader := NewLoader()
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("Expected port 9090 from env, got %d", cfg.Server.Port)
	}
}

func TestValidateServer(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *ServerConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &ServerConfig{
				Port:         8080,
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "invalid port (too high)",
			cfg: &ServerConfig{
				Port: 70000,
			},
			wantErr: true,
		},
		{
			name: "invalid port (zero)",
			cfg: &ServerConfig{
				Port: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid read timeout",
			cfg: &ServerConfig{
				Port:        8080,
				ReadTimeout: -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid write timeout",
			cfg: &ServerConfig{
				Port:         8080,
				WriteTimeout: -1 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateServer(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateServer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateEngine(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *EngineConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &EngineConfig{
				Workers:   "auto",
				QueueSize: 2000,
				Timeout:   20 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "invalid queue size",
			cfg: &EngineConfig{
				QueueSize: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid timeout",
			cfg: &EngineConfig{
				QueueSize: 2000,
				Timeout:   -1 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEngine(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEngine() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCache(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *CacheConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &CacheConfig{
				L1Size: 5000,
				L2TTL:  time.Hour,
			},
			wantErr: false,
		},
		{
			name: "negative L1 size",
			cfg: &CacheConfig{
				L1Size: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid L2 TTL",
			cfg: &CacheConfig{
				L1Size: 5000,
				L2TTL:  -1 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCache(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCache() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateNATS(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *NATSConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &NATSConfig{
				URLs: []string{"nats://localhost:4222"},
			},
			wantErr: false,
		},
		{
			name: "empty URLs",
			cfg: &NATSConfig{
				URLs: []string{},
			},
			wantErr: true,
		},
		{
			name: "nil URLs",
			cfg: &NATSConfig{
				URLs: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNATS(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateNATS() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateLogging(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *LoggingConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &LoggingConfig{
				Level:  "info",
				Format: "json",
			},
			wantErr: false,
		},
		{
			name: "invalid level",
			cfg: &LoggingConfig{
				Level: "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			cfg: &LoggingConfig{
				Level:  "info",
				Format: "invalid",
			},
			wantErr: true,
		},
		{
			name: "valid levels",
			cfg: &LoggingConfig{
				Level:  "debug",
				Format: "json",
			},
			wantErr: false,
		},
		{
			name: "valid formats",
			cfg: &LoggingConfig{
				Level:  "info",
				Format: "console",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLogging(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateLogging() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
