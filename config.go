package main

import "github.com/ilyakaznacheev/cleanenv"

type Config struct {
  Port      string          `yaml:"port" env:"HOUSTON_PORT" env-default:"8000"`
  Redis     RedisConfig     `yaml:"redis"`
  Password  string          `yaml:"password" env:"HOUSTON_PASSWORD"`
  Dashboard DashboardConfig `yaml:"dashboard"`
  Salt      string          // note: it is not recommended to set the salt yourself. It will be randomly generated
}

type DashboardConfig struct {
  Enabled bool   `yaml:"enabled" env:"HOUSTON_DASHBOARD" env-default:"true"`
  Src     string `yaml:"src" env:"HOUSTON_DASHBOARD_SRC" env-default:""`
}

type RedisConfig struct {
  Addr     string `yaml:"addr" env:"REDIS_ADDR" env-default:"localhost:6379"`
  Password string `yaml:"password" env:"REDIS_PASSWORD" env-default:""`
  DB       int    `yaml:"db" env:"REDIS_DB" env-default:"0"`
}

func LoadConfig(configPath string) *Config {
  var config Config
  if configPath == "" {
    err := cleanenv.ReadEnv(&config)
    if err != nil {
      panic(err)
    }
  } else {
    err := cleanenv.ReadConfig(configPath, &config)
    if err != nil {
      panic(err)
    }
  }
  return &config
}
