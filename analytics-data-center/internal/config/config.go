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
	OLTPstorages    []OLTPstorage `yaml:"oltp_connections"`
	TokenTTL        time.Duration `yaml:"token_ttl,omitempty"`
	GRPC            GRPCSetting   `yaml:"grpc" json:"grpc,omitempty"`
}

type GRPCSetting struct {
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

type OLTPstorage struct {
	Name string `yaml:"name"`
	Path string `yaml:"connection_string"`
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
