package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	SUNAT    SUNATConfig    `yaml:"sunat"`
	Security SecurityConfig `yaml:"security"`
}

type ServerConfig struct {
	Port         string `yaml:"port"`
	Host         string `yaml:"host"`
	ReadTimeout  int    `yaml:"read_timeout"`
	WriteTimeout int    `yaml:"write_timeout"`
}

type DatabaseConfig struct {
	Driver   string `yaml:"driver"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	SSLMode  string `yaml:"ssl_mode"`
	Schema   string `yaml:"schema"`
	MaxOpenConns int `yaml:"max_open_conns"`
	MaxIdleConns int `yaml:"max_idle_conns"`
}

type SUNATConfig struct {
	BaseURL     string `yaml:"base_url"`
	BetaURL     string `yaml:"beta_url"`
	RUC         string `yaml:"ruc"`
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
	Timeout     int    `yaml:"timeout"`
	MaxRetries  int    `yaml:"max_retries"`
	ForceRealSend bool  `yaml:"force_real_send"`
	FEBeta             string `yaml:"fe_beta"`
	FEHomologacion     string `yaml:"fe_homologacion"`
	FEProduccion       string `yaml:"fe_produccion"`
	FEConsultaCDR      string `yaml:"fe_consulta_cdr"`
	GuiaBeta           string `yaml:"guia_beta"`
	GuiaProduccion     string `yaml:"guia_produccion"`
	RetencionBeta      string `yaml:"retencion_beta"`
	RetencionProduccion string `yaml:"retencion_produccion"`
}

type SecurityConfig struct {
	CertificatePath string `yaml:"certificate_path"`
	PrivateKeyPath  string `yaml:"private_key_path"`
	CertificatePass string `yaml:"certificate_pass"`
	HashAlgorithm   string `yaml:"hash_algorithm"`
}

var AppConfig *Config

func LoadConfig() (*Config, error) {
	configFile := "config/app.yaml"
	if os.Getenv("CONFIG_FILE") != "" {
		configFile = os.Getenv("CONFIG_FILE")
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	// Validar configuraciones obligatorias
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}

	AppConfig = &config
	return &config, nil
}

func validateConfig(config *Config) error {
	if config.Server.Port == "" {
		return fmt.Errorf("El puerto del servidor es requerido")
	}
	if config.Database.Host == "" {
		return fmt.Errorf("El host es requerido")
	}
	if config.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}
	if config.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if config.SUNAT.RUC == "" {
		return fmt.Errorf("SUNAT RUC is required")
	}
	if config.Security.CertificatePath == "" {
		return fmt.Errorf("certificate path is required")
	}
	return nil
}

func GetConfig() *Config {
	return AppConfig
}