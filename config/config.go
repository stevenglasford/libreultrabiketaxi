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
	IMAP_Username string `yaml:"imap_username"`
	IMAP_Hostname string `yaml:"imap_hostname"`
	IMAP_Port int64 `yaml:"imap_port"`
	IMAP_Password string `yaml:"imap_password"`
	IMAP_Folder string `yaml:"imap_folder"`
	IMAP_SSL bool `yaml:"imap_ssl"`
	TEST_Receivers string `yaml:"test_receivers"`
	Db_Conn_Str string
	IMAP_Cert string `yaml:"imap_cert"`
	Cyclers_Api_Key string `yaml:"cyclers_api_key"`
	Cyclers_URL string `yaml:"cyclers_url"`
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
