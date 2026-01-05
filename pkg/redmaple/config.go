package redmaple

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port      int
	StaticDir string
	VendorDir string
	Timezone  string
}

func LoadConfig() Config {
	return Config{
		Port:      loadIntEnv("PORT", 6556),
		StaticDir: loadStrEnv("STATIC_DIR", "./static"),
		VendorDir: loadStrEnv("VENDOR_DIR", "./vendor"),
		Timezone:  loadStrEnv("TIMEZONE", "America/New_York"),
	}
}

func loadStrEnv(name, defaultVal string) string {
	val, ok := os.LookupEnv(name)
	if !ok {
		return defaultVal
	}
	return val
}

func loadBoolEnv(name string, defaultVal bool) bool {
	val, ok := os.LookupEnv(name)
	if !ok {
		return defaultVal
	}
	return strings.ToLower(val) == "true" || val == "1"
}

func loadIntEnv(name string, defaultVal int) int {
	valStr, ok := os.LookupEnv(name)
	if !ok {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}
