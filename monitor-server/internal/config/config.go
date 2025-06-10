package config

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v2"
)

// TDengineConfig 用于保存TDengine数据库配置
type TDengineConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Password string `yaml:"password"`
	DB       string `yaml:"db"`
}
type Grpc struct {
	Port string `yaml:"port"`
}
type Server struct {
	Port string `yaml:"port"`
}

// Config 用于保存所有配置项
type Config struct {
	TDengine TDengineConfig `yaml:"tdengine"`
	Redis    RedisConfig    `yaml:"redis"`
	Grpc     Grpc           `yaml:"grpc"`
	Server   Server         `yaml:"server"`
}

// getConfigPath 获取配置文件的路径
func getConfigPath() string {
	_, filename, _, ok := runtime.Caller(2) // 获取调用者的文件名
	if !ok {
		log.Fatal("无法获取运行时调用者信息")
	}

	// 获取当前文件所在的目录
	currentDir := filepath.Dir(filename)

	// 构建到项目根目录的相对路径
	configPath := filepath.Join(currentDir, "..", "..", "internal", "config", "configs", "config.yaml")

	// 将路径转换为绝对路径并简化路径
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		log.Fatalf("无法获取绝对路径: %v", err)
	}

	simplifiedPath := filepath.Clean(absPath)

	return simplifiedPath
}

// GetConfigPath 返回配置文件的路径
func GetConfigPath() string {
	return getConfigPath()
}

// LoadConfig 加载配置文件并返回 Config
func LoadConfig() (*Config, error) {
	configPath := GetConfigPath()
	yamlFile, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("读取配置文件失败: %v", err)
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Printf("解析配置文件失败: %v", err)
		return nil, err
	}

	return &config, nil
}
