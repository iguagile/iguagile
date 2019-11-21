package main

import (
	"log"
	"os"

	api "github.com/iguagile/iguagile-api"
)

func main() {
	api.Address = ":80"
	api.BaseUri = "/api/v1"
	api.RedisHost = os.Getenv("REDIS_HOST")
	api.MaxUser = 70
	api.Logger = log.New(os.Stdout, "iguagile-api ", log.Lshortfile)
	log.Fatal(api.Start())
}
