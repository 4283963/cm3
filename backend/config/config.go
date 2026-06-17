package config

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Station  StationConfig
}

type ServerConfig struct {
	Port         string
	WebSocketURL string
}

type DatabaseConfig struct {
	DSN string
}

type StationConfig struct {
	TotalMaxPower float64
	ChargerCount  int
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         ":8080",
			WebSocketURL: "ws://localhost:8080/ws",
		},
		Database: DatabaseConfig{
			DSN: "root:password@tcp(127.0.0.1:3306)/supercharger?charset=utf8mb4&parseTime=True&loc=Local",
		},
		Station: StationConfig{
			TotalMaxPower: 500.0,
			ChargerCount:  10,
		},
	}
}
