package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	App      AppConfig
	Database DatabaseConfig
	JWT      JWTConfig
	AWS      AWSConfig
	CORS     CORSConfig
	WS       WSConfig
	SMTP     SMTPConfig
	Log      LogConfig
}

type AppConfig struct {
	Env     string
	Port    string
	BaseURL string
}

type DatabaseConfig struct {
	Host         string
	Port         string
	Name         string
	User         string
	Password     string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
	ConnTimeout  time.Duration
	DSN          string
}

type JWTConfig struct {
	Secret         string
	AccessExpiry   time.Duration
	RefreshExpiry  time.Duration
}

type AWSConfig struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	S3Bucket        string
	S3BaseURL       string
	PresignExpiry   time.Duration
}

type CORSConfig struct {
	AllowedOrigins []string
}

type WSConfig struct {
	PingInterval time.Duration
	PongWait     time.Duration
	WriteWait    time.Duration
}

type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

type LogConfig struct {
	Level  string
	Format string
}

// Load reads .env (if present) then populates Config from environment variables.
func Load() (*Config, error) {
	// Try .env in cwd, then ../  (covers running from backend/ or from project root)
	for _, p := range []string{".env", "../.env", "../../.env"} {
		if err := godotenv.Load(p); err == nil {
			break
		}
	}

	cfg := &Config{}

	// App
	cfg.App = AppConfig{
		Env:     getEnv("APP_ENV", "development"),
		Port:    getEnv("APP_PORT", "8080"),
		BaseURL: getEnv("APP_BASE_URL", "http://localhost:8080"),
	}

	// Database
	maxOpen, _ := strconv.Atoi(getEnv("DB_MAX_OPEN_CONNS", "25"))
	maxIdle, _ := strconv.Atoi(getEnv("DB_MAX_IDLE_CONNS", "10"))
	connTimeout, _ := time.ParseDuration(getEnv("DB_CONN_TIMEOUT", "30s"))

	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbName := getEnv("DB_NAME", "pms3")
	dbUser := getEnv("DB_USER", "pms3_user")
	dbPass := mustEnv("DB_PASSWORD")
	dbSSL := getEnv("DB_SSLMODE", "disable")

	cfg.Database = DatabaseConfig{
		Host:         dbHost,
		Port:         dbPort,
		Name:         dbName,
		User:         dbUser,
		Password:     dbPass,
		SSLMode:      dbSSL,
		MaxOpenConns: maxOpen,
		MaxIdleConns: maxIdle,
		ConnTimeout:  connTimeout,
		DSN: fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=%s",
			dbUser, dbPass, dbHost, dbPort, dbName, dbSSL,
		),
	}

	// JWT
	accessExp, _ := time.ParseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m"))
	refreshExpStr := getEnv("JWT_REFRESH_EXPIRY", "168h")
	// Support shorthand "7d" → convert to hours
	refreshExp := parseDurationWithDays(refreshExpStr)
	cfg.JWT = JWTConfig{
		Secret:        mustEnv("JWT_SECRET"),
		AccessExpiry:  accessExp,
		RefreshExpiry: refreshExp,
	}

	// AWS
	presignExp, _ := time.ParseDuration(getEnv("AWS_S3_PRESIGN_EXPIRY", "15m"))
	cfg.AWS = AWSConfig{
		Region:          mustEnv("AWS_REGION"),
		AccessKeyID:     mustEnv("AWS_ACCESS_KEY_ID"),
		SecretAccessKey: mustEnv("AWS_SECRET_ACCESS_KEY"),
		S3Bucket:        mustEnv("AWS_S3_BUCKET"),
		S3BaseURL:       getEnv("AWS_S3_BASE_URL", ""),
		PresignExpiry:   presignExp,
	}

	// CORS
	cfg.CORS = CORSConfig{
		AllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")),
	}

	// WebSocket
	ping, _ := time.ParseDuration(getEnv("WS_PING_INTERVAL", "30s"))
	pong, _ := time.ParseDuration(getEnv("WS_PONG_WAIT", "60s"))
	write, _ := time.ParseDuration(getEnv("WS_WRITE_WAIT", "10s"))
	cfg.WS = WSConfig{
		PingInterval: ping,
		PongWait:     pong,
		WriteWait:    write,
	}

	// SMTP
	smtpPort, _ := strconv.Atoi(getEnv("SMTP_PORT", "587"))
	cfg.SMTP = SMTPConfig{
		Host:     getEnv("SMTP_HOST", ""),
		Port:     smtpPort,
		User:     getEnv("SMTP_USER", ""),
		Password: getEnv("SMTP_PASSWORD", ""),
		From:     getEnv("SMTP_FROM", ""),
	}

	// Logging
	cfg.Log = LogConfig{
		Level:  getEnv("LOG_LEVEL", "info"),
		Format: getEnv("LOG_FORMAT", "json"),
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return v
}

func splitCSV(s string) []string {
	var result []string
	for _, part := range splitTrim(s, ',') {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func splitTrim(s string, sep rune) []string {
	var out []string
	cur := ""
	for _, c := range s {
		if c == sep {
			out = append(out, cur)
			cur = ""
		} else {
			cur += string(c)
		}
	}
	out = append(out, cur)
	return out
}

// parseDurationWithDays handles "7d" in addition to standard Go durations.
func parseDurationWithDays(s string) time.Duration {
	if len(s) > 1 && s[len(s)-1] == 'd' {
		days := 0
		for _, c := range s[:len(s)-1] {
			if c >= '0' && c <= '9' {
				days = days*10 + int(c-'0')
			}
		}
		return time.Duration(days) * 24 * time.Hour
	}
	d, _ := time.ParseDuration(s)
	return d
}
