package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"time"
)

type Configuration struct {
	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`
	DispatcherConfig struct {
		AutoScalingGroupName string        `yaml:"autoScalingGroupName"`
		NumberOfRetries      int           `yaml:"numberOfRetries"`
		TickRefreshTime      time.Duration `yaml:"tickRefreshTime"`
	} `yaml:"dispatcher"`
	AWS struct {
		Region string `yaml:"region"`
	} `yaml:"aws"`
}

func LoadConfig(path string) (*Configuration, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Error while oppening configuration file: %w", err)
	}

	var cfg Configuration
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("Error while parssing yaml: %w", err)
	}
	fmt.Printf("Loaded configuration from %s\n", path)
	return &cfg, nil
}
