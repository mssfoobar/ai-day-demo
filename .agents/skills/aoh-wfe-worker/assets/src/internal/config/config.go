// Package config loads the worker's settings (Viper: flags > env > config.yaml >
// defaults). It is intentionally small — an activity worker needs only logging,
// a health port, and the Temporal connection. Add your own sections (external
// service URLs, credentials, etc.) here as your activities require them.
package config

import (
	"errors"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	Log      Log      `mapstructure:"log"`
	HTTP     HTTP     `mapstructure:"http"`
	Temporal Temporal `mapstructure:"temporal"`
}

type Log struct {
	Level string `mapstructure:"level"`
}

type HTTP struct {
	Port int `mapstructure:"port"`
}

// Temporal configures the connection to the Temporal cluster. TaskQueue MUST
// match the WFE workflow-engine's task queue (default "wfe"), or the engine's
// scheduled activities will never reach this worker.
type Temporal struct {
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	Namespace string `mapstructure:"namespace"`
	TaskQueue string `mapstructure:"task_queue"`
}

func Load() (*Config, error) {
	viper.SetDefault("log.level", "info")
	viper.SetDefault("http.port", 8080)
	viper.SetDefault("temporal.host", "localhost")
	viper.SetDefault("temporal.port", 7233)
	viper.SetDefault("temporal.namespace", "default")
	viper.SetDefault("temporal.task_queue", "wfe")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/app")

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, err
		}
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	pflag.String("log.level", "", "Log level [debug, info, warn, error]")
	pflag.Int("http.port", 0, "HTTP port for the health endpoints")
	pflag.String("temporal.host", "", "Temporal host")
	pflag.Int("temporal.port", 0, "Temporal port")
	pflag.String("temporal.namespace", "", "Temporal namespace")
	pflag.String("temporal.task_queue", "", "Temporal task queue (must match the WFE engine)")
	pflag.Parse()

	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
