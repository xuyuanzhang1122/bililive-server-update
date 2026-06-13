package config

import "os"

type Config struct {
	Addr          string
	DataDir       string
	AdminToken    string
	AdminUser     string
	AdminPassword string
	PublicBaseURL string
	GitHubRepo    string
}

func FromEnv() Config {
	return Config{
		Addr:          env("BLSU_ADDR", ":8090"),
		DataDir:       env("BLSU_DATA_DIR", "./data"),
		AdminToken:    env("BLSU_ADMIN_TOKEN", ""),
		AdminUser:     env("BLSU_ADMIN_USER", "xumy"),
		AdminPassword: env("BLSU_ADMIN_PASSWORD", "admin053164"),
		PublicBaseURL: env("BLSU_PUBLIC_BASE_URL", ""),
		GitHubRepo:    env("BLSU_GITHUB_REPO", "xuyuanzhang1122/bililive-go-UI"),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
