package config

import (
    "fmt"
    "os"
)

type DatabaseConfig struct {
    Host     string
    Port     string
    Name     string
    User     string
    Password string
    SSLMode  string
    SSLCert  string
    SSLKey   string
    SSLRootCert string
}

func LoadDatabaseConfig() *DatabaseConfig {
    sslMode := os.Getenv("DB_SSL_MODE")
    if sslMode == "" {
        sslMode = "require"  // Default to require SSL
    }
    
    return &DatabaseConfig{
        Host:        os.Getenv("DB_HOST"),
        Port:        os.Getenv("DB_PORT"),
        Name:        os.Getenv("DB_NAME"),
        User:        os.Getenv("DB_USER"),
        Password:    os.Getenv("DB_PASSWORD"),
        SSLMode:     sslMode,
        SSLCert:     os.Getenv("DB_SSL_CERT"),
        SSLKey:      os.Getenv("DB_SSL_KEY"),
        SSLRootCert: os.Getenv("DB_SSL_ROOT_CERT"),
    }
}

func (c *DatabaseConfig) ConnectionString() string {
    dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
        c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
    
    if c.SSLMode != "disable" {
        if c.SSLCert != "" {
            dsn += fmt.Sprintf(" sslcert=%s", c.SSLCert)
        }
        if c.SSLKey != "" {
            dsn += fmt.Sprintf(" sslkey=%s", c.SSLKey)
        }
        if c.SSLRootCert != "" {
            dsn += fmt.Sprintf(" sslrootcert=%s", c.SSLRootCert)
        }
    }
    
    return dsn
}
