package config

import (
	"os"
	"strconv"
	"time"
)

// Got it from like a tutorial. Will find links later.
func MustIntEnv(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// Adapted the one from above a bit.
// Citations:
// - I learnt about tickers here: https://dev.to/ankitmalikg/go-ticker-vs-timer-4glb
func MustDurationEnvMs(key string, defMs int) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return time.Duration(defMs) * time.Millisecond
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return time.Duration(defMs) * time.Millisecond
	}
	return time.Duration(n) * time.Millisecond
}
