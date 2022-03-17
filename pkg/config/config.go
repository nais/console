package config

import (
	"github.com/kelseyhightower/envconfig"
)

type GitHub struct {
	AppId             int64  `envconfig:"CONSOLE_GITHUB_APP_ID"`
	AppInstallationId int64  `envconfig:"CONSOLE_GITHUB_APP_INSTALLATION_ID"`
	Organization      string `envconfig:"CONSOLE_GITHUB_ORGANIZATION"`
	PrivateKeyPath    string `envconfig:"CONSOLE_GITHUB_PRIVATE_KEY_PATH"`
}

type Google struct {
	CredentialsFile string `envconfig:"CONSOLE_GOOGLE_CREDENTIALS_FILE"`
	DelegatedUser   string `envconfig:"CONSOLE_GOOGLE_DELEGATED_USER"`
	Domain          string `envconfig:"CONSOLE_GOOGLE_DOMAIN"`
}

type NaisDeploy struct {
	Endpoint     string `envconfig:"CONSOLE_NAIS_DEPLOY_ENDPOINT"`
	ProvisionKey string `envconfig:"CONSOLE_NAIS_DEPLOY_PROVISION_KEY"`
}

type Config struct {
	GitHub        GitHub
	Google        Google
	NaisDeploy    NaisDeploy
	DatabaseURL   string `envconfig:"CONSOLE_DATABASE_URL"`
	ListenAddress string `envconfig:"CONSOLE_LISTEN_ADDRESS"`
}

func Defaults() *Config {
	return &Config{
		DatabaseURL:   "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable",
		ListenAddress: "127.0.0.1:3000",
		NaisDeploy: NaisDeploy{
			Endpoint: "http://localhost:8080/api/v1/provision",
		},
	}
}

func New() (*Config, error) {
	cfg := Defaults()

	err := envconfig.Process("", cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}