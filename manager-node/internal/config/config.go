package config

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"log"
	"time"
)

type Config struct {
	PublicPort  string `envconfig:"PUBLIC_PORT" required:"true"`
	PrivatePort string `envconfig:"PRIVATE_PORT" required:"true"`

	CheckHealthInterval time.Duration `envconfig:"HEALTH_CHECK_INTERVAL" required:"true"`
}

func LoadConfig() *Config {

	for _, fileName := range []string{".env.local", ".env"} {
		err := godotenv.Load(fileName)
		if err != nil {
			log.Println("[CONFIG][ERROR]:", err)
		}
	}

	cfg := Config{}

	err := envconfig.Process("", &cfg)
	if err != nil {
		log.Fatalln("[CONFIG][ERROR]:", err)
	}

	cfg.PrintConfig()

	return &cfg
}

func (c *Config) PrintConfig() {
	log.Println("===================== CONFIG =====================")
	log.Println("_____________SERVER____________ ")
	log.Println("PUBLIC_PORT.................... ", c.PublicPort)
	log.Println("PRIVATE_PORT................... ", c.PrivatePort)
	log.Println("HEALTH_CHECK_INTERVAL.......... ", c.CheckHealthInterval)

	log.Println("==================================================")
}
