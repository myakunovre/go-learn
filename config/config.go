package config

var Config = struct {
	ServerPort  string
	DatabaseURL string
}{
	ServerPort:  "8080",
	DatabaseURL: "host=localhost port=5432 user=user password=password dbname=products_db sslmode=disable",
}
