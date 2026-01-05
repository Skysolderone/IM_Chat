package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

var globalConfig *Config

type Config struct {
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`

	Database struct {
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		DBName   string `yaml:"dbname"`
		SSLMode  string `yaml:"sslmode"`
		Timezone string `yaml:"timezone"`

		MaxOpenConns    int           `yaml:"max_open_conns"`
		MaxIdleConns    int           `yaml:"max_idle_conns"`
		ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
		ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"`
	} `yaml:"database"`

	JWT struct {
		Secret string        `yaml:"secret"`
		Expire time.Duration `yaml:"expire"`
	} `yaml:"jwt"`
}

func LoadConfig(configPath string) error {
	if configPath == "" {
		configPath = "config/config.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	globalConfig = config
	return nil
}

func GetConfig() *Config {
	if globalConfig == nil {
		panic("配置未初始化，请先调用 LoadConfig")
	}
	return globalConfig
}

func GetServerPort() string {
	return GetConfig().Server.Port
}

func GetDatabaseConfig() *Config {
	return GetConfig()
}

func GetJWTSecret() string {
	secret := GetConfig().JWT.Secret
	if secret == "" {
		return "secret key"
	}
	return secret
}

func GetJWTExpire() time.Duration {
	expire := GetConfig().JWT.Expire
	if expire == 0 {
		return time.Hour
	}
	return expire
}
