package config

import (
	"github.com/spf13/viper"
	"fmt"
	"io/ioutil"
	"log"
	"gopkg.in/yaml.v2"
)

type Config struct {
	SMTP_Username string `yaml:"smtp_username"`
	SMTP_Token string `yaml:"smtp_token"`
	SMTP_Server string `yaml:"smtp_server"`
	SMTP_Port int64 `yaml:"smtp_port"`
	TEST_Receivers string
	Db_Conn_Str string
	Rabbit_Url string
	Admin_Channel_Chat_Id int64
	Public_Channel_Chat_Id int64
}

var config Config

func C() *Config {
	return &config
}

func Init(file string) {
	viper.SetConfigName(file)
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Error in config file: %s", err))
	}

	viper.Unmarshal(&config)
}
