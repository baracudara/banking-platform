package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Environment names. They live here rather than in main because config is
// what decides which behaviour each environment gets.
const (
	EnvLocal = "local"
	EnvDev   = "dev"
	EnvProd  = "prod"
)

type Config struct {
	Env      string   `yaml:"env" env:"ENV" env-default:"local"`
	Postgres Postgres `yaml:"postgres"`
	GRPC     GRPC     `yaml:"grpc"`
}

type Postgres struct {
	Host     string `yaml:"host" env:"POSTGRES_HOST" env-default:"localhost"`
	Port     int    `yaml:"port" env:"POSTGRES_PORT" env-default:"5432"`
	User     string `yaml:"user" env:"POSTGRES_USER" env-default:"banking"`
	Password string `yaml:"password" env:"POSTGRES_PASSWORD" env-default:"banking"`
	DBName   string `yaml:"dbname" env:"POSTGRES_DB" env-default:"account"`

	// Pool sizing is a config value, not a constant, because the right
	// number depends on the deployment: it must be balanced against
	// Postgres' own max_connections across every service and replica.
	MinConns int32 `yaml:"min_conns" env-default:"2"`
	MaxConns int32 `yaml:"max_conns" env-default:"10"`

	ConnectTimeout time.Duration `yaml:"connect_timeout" env-default:"5s"`
}

type GRPC struct {
	Port int `yaml:"port" env:"GRPC_PORT" env-default:"50051"`

	// Upper bound on how long GracefulStop may drain in-flight RPCs before
	// the server is killed outright. Without a bound, a single client that
	// never closes its stream keeps the process alive forever and the
	// orchestrator eventually SIGKILLs it mid-transaction.
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env-default:"10s"`
}

// DSN builds the Postgres connection string. It lives here so that the
// repository layer never has to know the Config type exists - it just
// receives a string, which keeps the dependency arrow pointing one way.
func (p Postgres) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		p.User, p.Password, p.Host, p.Port, p.DBName,
	)
}

// ReflectionEnabled reports whether the gRPC server should serve its own
// schema. Reflection is how grpcurl/buf curl call a service without being
// given the .proto files, which is worth a lot while developing - but it
// also hands anyone who can reach the port a complete map of the API, so it
// stays off in production.
func (c *Config) ReflectionEnabled() bool {
	return c.Env != EnvProd
}

func MustLoad() *Config {
	cfg, err := Load(fetchConfigPath())
	if err != nil {
		panic(err)
	}
	return cfg
}

// Load reads the config file if one was given, then lets environment
// variables override whatever it contains. An empty path is not an error:
// in a container there is usually no file at all, only env vars.
func Load(path string) (*Config, error) {
	var cfg Config

	if path == "" {
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			return nil, fmt.Errorf("read config from env: %w", err)
		}
		return &cfg, nil
	}

	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("config file %q: %w", path, err)
	}
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	return &cfg, nil
}

// fetchConfigPath resolves the config path from --config, then CONFIG_PATH,
// then nothing. The flag default is deliberately empty: giving it a real
// default would make the value never empty, and the environment fallback
// below would become dead code that silently never runs.
func fetchConfigPath() string {
	var path string
	flag.StringVar(&path, "config", "", "path to config file")
	flag.Parse()

	if path == "" {
		path = os.Getenv("CONFIG_PATH")
	}
	return path
}
