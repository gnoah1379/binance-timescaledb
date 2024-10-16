package main

type Config struct {
	DBName string `json:"db_name"`
	DBUser string `json:"db_user"`
	DBPwd  string `json:"db_pwd"`
	DBHost string `json:"db_host"`
	DBPort int    `json:"db_port"`
}
