package main

import (
	"encoding/json"
	"github.com/ilyakaznacheev/cleanenv"
	"time"
)

type Config struct {
	Port      string          `yaml:"port" env:"HOUSTON_PORT" env-default:"8000" json:"port"`
	Redis     RedisConfig     `yaml:"redis" json:"redis"`
	Password  string          `yaml:"password" env:"HOUSTON_PASSWORD" json:"password"`
	Dashboard DashboardConfig `yaml:"dashboard" json:"dashboard"`
	TLS       TLSConfig       `yaml:"tls" json:"tls"`
	//MissionExpiry time.Duration   `yaml:"mission_expiry" env:"HOUSTON_MISSION_EXPIRY" env-default:"720h"` // 30 days
	MissionExpiry time.Duration `yaml:"mission_expiry" env:"HOUSTON_MISSION_EXPIRY" env-default:"1s"`
	//MemoryLimitMiB int64 `yaml:"memory_limit_mib" env:"HOUSTON_MEMORY_LIMIT_MIB" env-default:"1024"`
	MemoryLimitMiB int64  `yaml:"memory_limit_mib" env:"HOUSTON_MEMORY_LIMIT_MIB" env-default:"0"`
	Salt           string `json:"-"` // note: it is not recommended to set the salt yourself. It will be randomly generated
}

type DashboardConfig struct {
	Enabled bool   `yaml:"enabled" env:"HOUSTON_DASHBOARD" env-default:"true" json:"enabled"`
	Src     string `yaml:"src" env:"HOUSTON_DASHBOARD_SRC" env-default:"" json:"src"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr" env:"REDIS_ADDR" env-default:"localhost:6379" json:"addr"`
	Password string `yaml:"password" env:"REDIS_PASSWORD" env-default:"" json:"password"`
	DB       int    `yaml:"db" env:"REDIS_DB" env-default:"0" json:"db"`
}

type TLSConfig struct {
	Auto     bool   `yaml:"auto" env:"TLS_AUTO" env-default:"true" json:"auto"`
	Host     string `yaml:"host" env:"TLS_HOST" env-default:"" json:"host"`
	CertFile string `yaml:"certFile" env:"TLS_CERT_FILE" env-default:"cert.pem" json:"certFile"`
	KeyFile  string `yaml:"keyFile" env:"TLS_KEY_FILE" env-default:"key.pem" json:"keyFile"`
}

func LoadConfig(configPath string) Config {
	var config Config
	if configPath == "" {
		log.Debug("No config file provided. Loading configuration from environment variables")
		err := cleanenv.ReadEnv(&config)
		if err != nil {
			panic(err)
		}
	} else {
		log.Debugf("Loading configuration from %s", configPath)
		err := cleanenv.ReadConfig(configPath, &config)
		if err != nil {
			panic(err)
		}
	}

	configJson, _ := json.Marshal(config)
	log.Debug("Configuration Loaded: ", string(configJson))

	return config
}
