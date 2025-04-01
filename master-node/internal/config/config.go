package config

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"log"
	"strings"
	"time"
)

type Config struct {
	UUID        string
	PublicPort  string `envconfig:"PUBLIC_PORT" required:"true"`
	PrivatePort string `envconfig:"PRIVATE_PORT" required:"true"`

	ManagerURL        string `envconfig:"MANAGER_URL" required:"true"`
	ManagerRegPath    string `envconfig:"MANAGER_REG_PATH" required:"true"`
	ManagerAddPath    string `envconfig:"MANAGER_TASK_ADD" required:"true"`
	ManagerClosePath  string `envconfig:"MANAGER_TASK_CLOSE" required:"true"`
	ManagerStatusPath string `envconfig:"MANAGER_TASK_STATUS" required:"true"`

	TaskScriptPath      string `envconfig:"TASK_SCRIPT_PATH" required:"true"`
	TaskComputeFuncName string `envconfig:"TASK_COMPUTE_FUNC_NAME" required:"true"`
	TaskFuncArgs        string `envconfig:"TASK_FUNC_ARGS" required:"true"`
	TaskArgs            []string

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

	cfg.TaskArgs = strings.Split(cfg.TaskFuncArgs, ",")

	cfg.PrintConfig()

	return &cfg
}

func (c *Config) PrintConfig() {
	log.Println("===================== CONFIG =====================")
	log.Println("_____________SERVER____________ ")
	log.Println("PUBLIC_PORT.................... ", c.PublicPort)
	log.Println("PRIVATE_PORT................... ", c.PrivatePort)
	log.Println("_____________TASK____________ ")
	log.Println("SCRIPT_PATH................... ", c.TaskScriptPath)
	log.Println("COMPUTE_FUNC_NAME............. ", c.TaskComputeFuncName)
	log.Println("INPUT_ARGS.................... ", c.TaskFuncArgs)

	log.Println("==================================================")
}
