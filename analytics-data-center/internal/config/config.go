package config

import (
	"flag"
	"os"
	"time"

	"github.com/numbergroup/cleanenv"
)

type Config struct {
	Env             string        `yaml:"env" env-default:"local" json:"env,omitempty"`
	StoragePath     string        `yaml:"storage_path" env-required:"true" json:"storage_path,omitempty"`
	OLTPDataBase    string        `yaml:"oltp_db" json:"oltp_db,omitempty"`
	OLTPStoragePath string        `yaml:"oltp_db_path" json:"oltp_db_path,omitempty"`
	DWHDataBase     string        `yaml:"dwh_db" json:"dwh_db,omitempty"`
	DWHStoragePath  string        `yaml:"dwh_db_path" json:"dwh_db_path,omitempty"`
	OLTPstorages    []OLTPstorage `yaml:"oltp_connections" json:"olt_pstorages,omitempty"`
	TokenTTL        time.Duration `yaml:"token_ttl,omitempty" json:"token_ttl,omitempty"`
	LogLang         string        `yaml:"log_lang" env-default:"ru" json:"log_lang,omitempty"`
	GRPC            GRPCSetting   `yaml:"grpc" json:"grpc,omitempty"`
	Kafka           KafkaSetting  `yaml:"kafka" json:"kafka,omitempty"`
	KafkaConnect    string        `yaml:"kafka_connect" json:"kafka_connect,omitempty"`
	SMTP            SMTPSetting   `yaml:"smtp_setting" json:"smtp_setting,omitempty"`
}

type SMTPSetting struct {
	Host       string `yaml:"host" env-default:"smtp.gmail.com" json:"host,omitempty"`
	Port       int    `yaml:"port" env-default:"587" json:"port,omitempty"`
	UserName   string `yaml:"username" env-default:"your_email@gmail.com" json:"username,omitempty"`
	Password   string `yaml:"password" env-default:"your_password" json:"password,omitempty"`
	AdminEmail string `yaml:"admin_email" env-default:"your_email@gmail.com" json:"admin_email,omitempty"`
	FromEmail  string `yaml:"from_email" env-default:"your_email@gmail.com" json:"from_email,omitempty"`
}

type GRPCSetting struct {
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

type OLTPstorage struct {
	Name      string `yaml:"name"`
	Path      string `yaml:"connection_string"`
	PathKafka string `yaml:"connection_string_kafka"`
}

type KafkaSetting struct {
	BootstrapServers string `yaml:"bootstrap.servers"`
	Acks             string `yaml:"acks"`
	ClientId         string `yaml:"client_id"`
	EnableAutoCommit string `yaml:"enable.auto.commit"`
	AutoOffsetReset  string `yaml:"auto.offset.reset"`
	SessionTimeoutMs string `yaml:"session.timeout.ms"`
	GroupId          string `yaml:"group.id"`
}

func MustLoad() *Config {
	path := fetchConfigPath()
	if path == "" {
		panic("config path is empty")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic("config file does not exist" + path)
	}

	var cfg Config

	err := cleanenv.ReadConfig(path, &cfg)
	if err != nil {
		panic("failed to read config")
	}

	return &cfg
}

func fetchConfigPath() string {
	var res string
	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}
