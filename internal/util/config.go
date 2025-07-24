package util

import (
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

//nolint:gochecknoglobals // here its ok
var once sync.Once

func init() {
	once.Do(func() {
		if err := godotenv.Load(".env"); err != nil {
			log.Printf("Warning: could not load .env file: %v", err)
		}
	})
}

const (
	defaultServerAddr      = "localhost:8080"
	defaultWriteTimeout    = 10 * time.Second
	defaultReadTimeout     = 10 * time.Second
	defaultIdleTimeout     = 30 * time.Second
	defaultGracefulTimeout = 5 * time.Second

	defaultAccessTTL  = 15 * time.Minute
	defaultRefreshTTL = 24 * time.Hour

	defaultRateLimit     = 100
	defaultRateInterval  = 1 * time.Minute
	defaultRateBlockTime = 5 * time.Minute

	TokenPartsExpected = 2
	RawTokenLength     = 32
	JWTLeeWay          = 5 * time.Second
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

type RateLimiterConfig struct {
	Limit     int
	Interval  time.Duration
	BlockTime time.Duration
}

func NewRateLimiterConfig() *RateLimiterConfig {
	limitStr := os.Getenv("RATE_LIMIT_LIMIT")
	limit := defaultRateLimit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		} else {
			log.Printf("Invalid RATE_LIMIT_LIMIT: %s, using default %d", limitStr, defaultRateLimit)
		}
	}

	interval := parseDurationOrDefault("RATE_LIMIT_INTERVAL", defaultRateInterval)
	blockTime := parseDurationOrDefault("RATE_LIMIT_BLOCK_TIME", defaultRateBlockTime)

	return &RateLimiterConfig{
		Limit:     limit,
		Interval:  interval,
		BlockTime: blockTime,
	}
}

func GetWebhookURL() string {
	return os.Getenv("WEBHOOK_URL")
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
