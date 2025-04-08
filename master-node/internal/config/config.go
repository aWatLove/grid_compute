package config

import (
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"log"
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

	TaskScriptComputePath  string `envconfig:"TASK_SCRIPT_COMPUTE_PATH" required:"true"`
	TaskFuncNameCompute    string `envconfig:"TASK_COMPUTE_FUNC_NAME_COMPUTE" required:"true"`
	TaskScriptGeneratePath string `envconfig:"TASK_SCRIPT_GENERATE_PATH" required:"true"`
	TaskFuncNameGenerate   string `envconfig:"TASK_COMPUTE_FUNC_NAME_GENERATE" required:"true"`

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

	cfg.UUID = uuid.NewString()

	cfg.PrintConfig()

	return &cfg
}

func (c *Config) PrintConfig() {
	log.Println("===================== CONFIG =====================")
	log.Println("_____________SERVER____________ ")
	log.Println("UUID................................. ", c.UUID)
	log.Println("PUBLIC_PORT.......................... ", c.PublicPort)
	log.Println("PRIVATE_PORT......................... ", c.PrivatePort)
	log.Println("_____________TASK____________ ")
	log.Println("TASK_SCRIPT_COMPUTE_PATH............. ", c.TaskScriptComputePath)
	log.Println("TASK_COMPUTE_FUNC_NAME_COMPUTE....... ", c.TaskFuncNameCompute)
	log.Println("TASK_SCRIPT_GENERATE_PATH............ ", c.TaskScriptGeneratePath)
	log.Println("TASK_COMPUTE_FUNC_NAME_GENERATE...... ", c.TaskFuncNameGenerate)

	log.Println("==================================================")
}
