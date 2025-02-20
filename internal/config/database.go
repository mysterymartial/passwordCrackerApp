package config

import "fmt"

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

func NewDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Host:     "localhost",
		Port:     "3306",
		User:     "root",
		Password: "Mystery@ sax",
		DBName:   "password_cracker",
	}
}

func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.DBName,
	)
}
