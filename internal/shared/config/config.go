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
	Port int `env:"port"`
}

type Postgres struct {
	Host     string `env:"host"`
	Port     int    `env:"port"`
	User     string `env:"user"`
	Password string `env:"password"`
	Database string `env:"database"`
}

type Redis struct {
	URL string `env:"url"`
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
	Secret string `env:"secret"`
}

type Slack struct {
	Token   string `env:"token"`
	Channel string `env:"channel"`
}

type Limit struct {
	Status bool `env:"status"`
}

func Environments() {
	Once.Do(func() {
		cfg = new(Config)
		if err := config.GetConfigFromEnv(cfg); err != nil {
			log.Panicf("error parsing enviroment vars \n%v", err)
		}
	})
}
