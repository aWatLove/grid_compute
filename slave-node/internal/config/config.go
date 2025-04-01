package config

import (
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"log"
)

type Config struct {
	UUID           string `json:"UUID"`
	ManagerURL     string `envconfig:"MANAGER_URL" required:"true"`
	ManagerRegPath string `envconfig:"MANAGER_REG_PATH" required:"true"`

	PublicPort  string `envconfig:"PUBLIC_PORT" required:"true"`
	PrivatePort string `envconfig:"PRIVATE_PORT" required:"true"`
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

	cfg.UUID = uuid.NewString()

	cfg.PrintConfig()

	return &cfg
}

func (c *Config) PrintConfig() {
	log.Println("===================== CONFIG =====================")
	log.Println("UUID.......................... ", c.UUID)
	log.Println("_____________MASTER____________ ")
	log.Println("MASTER_URL.................... ", c.ManagerURL)
	log.Println("_____________SERVER____________ ")
	log.Println("PUBLIC_PORT.................... ", c.PublicPort)
	log.Println("PRIVATE_PORT.................... ", c.PrivatePort)

	log.Println("==================================================")
}
