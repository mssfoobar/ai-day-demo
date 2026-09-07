package config

import (
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	Log  Log  `mapstructure:"log"`
	HTTP HTTP `mapstructure:"http"`
	SQL  SQL  `mapstructure:"sql"`
	IAMS IAMS `mapstructure:"iams"`
}

type Log struct {
	Level string `mapstructure:"level"`
}

type HTTP struct {
	Port           int      `mapstructure:"port"`
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

type SQL struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	PluginName      string        `mapstructure:"plugin_name"`
	DatabaseName    string        `mapstructure:"database_name"`
	SchemaName      string        `mapstructure:"schema_name"`
	MaxConns        int           `mapstructure:"max_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
	SslMode         string        `mapstructure:"ssl_mode"`
}

type IAMS struct {
	KeycloakHost         string `mapstructure:"keycloak_host"`
	KeycloakPort         int    `mapstructure:"keycloak_port"`
	KeycloakClientID     string `mapstructure:"keycloak_client_id"`
	KeycloakClientSecret string `mapstructure:"keycloak_client_secret"`
	KeycloakRealm        string `mapstructure:"keycloak_realm"`
	AasHost              string `mapstructure:"aas_host"`
	AasPort              int    `mapstructure:"aas_port"`
}

func Load() (*Config, error) {
	viper.SetDefault("log.level", "info")
	viper.SetDefault("http.port", 8080)
	viper.SetDefault("http.allowed_origins", []string{})
	viper.SetDefault("sql.host", "localhost")
	viper.SetDefault("sql.port", 5432)
	viper.SetDefault("sql.user", "postgres")
	viper.SetDefault("sql.plugin_name", "postgres")
	viper.SetDefault("sql.schema_name", "public")
	viper.SetDefault("sql.ssl_mode", "disable")
	viper.SetDefault("sql.max_conns", 10)
	viper.SetDefault("sql.max_idle_conns", 5)
	viper.SetDefault("sql.max_conn_lifetime", 5*time.Minute)
	viper.SetDefault("iams.keycloak_host", "http://iams-keycloak")
	viper.SetDefault("iams.keycloak_port", 8080)
	viper.SetDefault("iams.keycloak_realm", "aoh")
	viper.SetDefault("iams.aas_host", "http://iams-aas")
	viper.SetDefault("iams.aas_port", 8080)

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
	pflag.Int("http.port", 0, "HTTP port")
	pflag.StringSlice("http.allowed_origins", []string{}, "Allowed CORS origins (comma-separated)")
	pflag.String("sql.host", "", "Database host")
	pflag.Int("sql.port", 0, "Database port")
	pflag.String("sql.user", "", "Database user")
	pflag.String("sql.password", "", "Database password")
	pflag.String("sql.plugin_name", "", "Database plugin [postgres]")
	pflag.String("sql.database_name", "", "Database name")
	pflag.String("sql.schema_name", "", "Database schema name")
	pflag.Int("sql.max_conns", 0, "Max connections")
	pflag.Int("sql.max_idle_conns", 0, "Max idle connections")
	pflag.Duration("sql.max_conn_lifetime", 0, "Max connection lifetime")
	pflag.String("sql.ssl_mode", "", "SSL mode")
	pflag.String("iams.keycloak_host", "", "IAMS Keycloak host")
	pflag.Int("iams.keycloak_port", 0, "IAMS Keycloak port")
	pflag.String("iams.keycloak_client_id", "", "IAMS Keycloak client ID")
	pflag.String("iams.keycloak_client_secret", "", "IAMS Keycloak client secret")
	pflag.String("iams.keycloak_realm", "", "IAMS Keycloak realm")
	pflag.String("iams.aas_host", "", "IAMS AAS host")
	pflag.Int("iams.aas_port", 0, "IAMS AAS port")
	pflag.Parse()

	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			stringToSliceHookFunc(),
		),
	)); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func stringToSliceHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Kind, t reflect.Kind, data interface{}) (interface{}, error) {
		if f != reflect.String || t != reflect.Slice {
			return data, nil
		}
		str, ok := data.(string)
		if !ok {
			return data, nil
		}
		if str == "" {
			return []string{}, nil
		}
		return strings.Split(str, ","), nil
	}
}
