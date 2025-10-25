package config

import (
	"log"
	"sync"

	"github.com/andresxlp/gosuite/config"
)

var (
	Once sync.Once
	cfg  *Config
)

func Get() *Config {
	if cfg == nil {
		log.Panic("Configuration has not yet been initialized")
	}
	return cfg
}

type Config struct {
	Server      Server      `env:"server"`
	Postgres    Postgres    `env:"postgres"`
	Redis       Redis       `env:"redis"`
	RedisStream RedisStream `env:"redis_stream"`
	JWT         JWT         `env:"jwt"`
	Slack       Slack       `env:"slack"`
	Limit       Limit       `env:"limit"`
}

type Server struct {
	Port int `validate:"required" env:"port"`
}

type Postgres struct {
	Host     string `validate:"required" env:"host"`
	Port     int    `validate:"required" env:"port"`
	User     string `validate:"required" env:"user"`
	Password string `validate:"required" env:"password"`
	Database string `validate:"required" env:"database"`
}

type Redis struct {
	URL string `validate:"required" env:"url"`
}

type RedisStream struct {
	Host       string `validate:"required" env:"host"`
	Port       int    `validate:"required" env:"port"`
	StreamName string `validate:"required" env:"stream_name"`
	Db         int    `validate:"min=0" env:"db"`
	User       string `validate:"required" env:"user"`
	Password   string `validate:"required" env:"password"`
}

type JWT struct {
	Secret string `validate:"required" env:"secret"`
}

type Slack struct {
	Token   string `env:"token"`
	Channel string `env:"channel"`
}

type Limit struct {
	Status bool `validate:"required" env:"status"`
}

func Environments() {
	Once.Do(func() {
		cfg = new(Config)
		if err := config.GetConfigFromEnv(cfg); err != nil {
			log.Panicf("error parsing enviroment vars \n%v", err)
		}
	})
}
