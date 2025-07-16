package util

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Failed to load .env: %v", err)
	}
}

const (
	defaultServerAddr      = "localhost:8080"
	defaultWriteTimeout    = 10 * time.Second
	defaultReadTimeout     = 10 * time.Second
	defaultIdleTimeout     = 30 * time.Second
	defaultGracefulTimeout = 5 * time.Second

	defaultAccessTTL  = 15 * time.Minute
	defaultRefreshTTL = 24 * time.Hour
)

type ServerConfig struct {
	ServerAddr      string
	WriteTimeout    time.Duration
	ReadTimeout     time.Duration
	IdleTimeout     time.Duration
	GracefulTimeout time.Duration
	
}

func NewServerConfig() *ServerConfig {
	addr := os.Getenv("SERVER_ADDRESS")
	if addr == "" {
		addr = defaultServerAddr
	}

	

	return &ServerConfig{
		ServerAddr:      addr,
		WriteTimeout:    parseDurationOrDefault("WRITE_TIMEOUT", defaultWriteTimeout),
		ReadTimeout:     parseDurationOrDefault("READ_TIMEOUT", defaultReadTimeout),
		IdleTimeout:     parseDurationOrDefault("IDLE_TIMEOUT", defaultIdleTimeout),
		GracefulTimeout: parseDurationOrDefault("GRACEFUL_TIMEOUT", defaultGracefulTimeout),
	}
}

type TokenConfig struct {
	JwtSecretKey []byte
	AccessTTL    time.Duration
	RefreshTTL   time.Duration
}

func NewTokenConfig() *TokenConfig {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET is not set")
	}
	return &TokenConfig{
		JwtSecretKey: []byte(secret),
		AccessTTL:    parseDurationOrDefault("ACCESS_TOKEN_TTL", defaultAccessTTL),
		RefreshTTL:   parseDurationOrDefault("REFRESH_TOKEN_TTL", defaultRefreshTTL),
	}
}

func parseDurationOrDefault(varName string, def time.Duration) time.Duration {
	if v := os.Getenv(varName); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		log.Printf("Invalid duration in %s: %s, using default %s", varName, v, def)
	}
	return def
}
