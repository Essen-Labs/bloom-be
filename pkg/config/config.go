package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config contain configuration of db for migrator
// config var < env < command flag
type Config struct {
	ServiceName     string
	BaseURL         string
	Port            string
	Env             string
	AllowedOrigins  string
	DBHost          string
	DBPort          string
	DBUser          string
	DBName          string
	DBPass          string
	DBSSLMode       string
	AkashAPIKey     string
	GptAPIKey       string
	FrontendBaseURL string
}

// GetCORS in config
func (c *Config) GetCORS() []string {
	cors := strings.Split(c.AllowedOrigins, ";")
	rs := []string{}
	for idx := range cors {
		itm := cors[idx]
		if strings.TrimSpace(itm) != "" {
			rs = append(rs, itm)
		}
	}
	return rs
}

// Loader load config from reader into Viper
type Loader interface {
	Load(viper.Viper) (*viper.Viper, error)
}

// generateConfigFromViper generate config from viper data
func generateConfigFromViper(v *viper.Viper) Config {

	return Config{
		Port:           v.GetString("PORT"),
		BaseURL:        v.GetString("BASE_URL"),
		ServiceName:    v.GetString("SERVICE_NAME"),
		Env:            v.GetString("ENV"),
		AllowedOrigins: v.GetString("ALLOWED_ORIGINS"),

		DBHost:    v.GetString("DB_HOST"),
		DBPort:    v.GetString("DB_PORT"),
		DBUser:    v.GetString("DB_USER"),
		DBName:    v.GetString("DB_NAME"),
		DBPass:    v.GetString("DB_PASS"),
		DBSSLMode: v.GetString("DB_SSL_MODE"),

		AkashAPIKey:     v.GetString("AKASH_API_KEY"),
		GptAPIKey:       v.GetString("GPT_API_KEY"),
		FrontendBaseURL: v.GetString("FRONTEND_BASE_URL"),
	}
}

// DefaultConfigLoaders is default loader list
func DefaultConfigLoaders() []Loader {
	loaders := []Loader{}
	fileLoader := NewFileLoader(".env", ".")
	loaders = append(loaders, fileLoader)
	loaders = append(loaders, NewENVLoader())

	return loaders
}

// LoadConfig load config from loader list
func LoadConfig(loaders []Loader) Config {
	v := viper.New()
	v.SetDefault("PORT", "8080")
	v.SetDefault("ENV", "local")

	for idx := range loaders {
		newV, err := loaders[idx].Load(*v)

		if err == nil {
			v = newV
		}
	}
	return generateConfigFromViper(v)
}

// GetShutdownTimeout get shutdown time out
func (c *Config) GetShutdownTimeout() time.Duration {
	return 10 * time.Second
}
