package config

import (
	"github.com/spf13/viper"
	"fmt"
)

type Config struct {
	SMTP_Username string `yaml:"smtp_username"`
	SMTP_Token string `yaml:"smtp_token"`
	SMTP_Server string `yaml:"smtp_server"`
	SMTP_Port int64 `yaml:"smtp_port"`
	TEST_Receivers string `yaml:"test_receivers"`
	Db_Conn_Str string
	Rabbit_Url string
}

var config Config

func C() *Config {
	return &config
}

func Init(file string) {
	viper.SetConfigName(file)
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetConfigType("yaml")

	
	
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("Error in config file: %s", err))
	}

	// smtp_username := viper.GetString("smtp_username")
	// smtp_token := viper.GetString("smtp_token")
	// smtp_server := viper.GetString("smtp_server")
	// smtp_port := viper.GetInt64("smtp_port")
	// test_receivers := viper.GetString("test_receivers")
	// db_conn_str := viper.GetString("db_conn_str")
	// rabbit_url := viper.GetString("rabbit_url")

	viper.Unmarshal(&config)
}
