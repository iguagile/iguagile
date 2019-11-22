package main

import (
	"log"
	"os"

	api "github.com/iguagile/iguagile-api"
)

func main() {
	apiServer := api.NewRoomAPIServer()
	apiServer.Address = ":80"
	apiServer.BaseUri = "/api/v1"
	apiServer.RedisHost = os.Getenv("REDIS_HOST")
	apiServer.MaxUser = 70
	apiServer.Logger = log.New(os.Stdout, "iguagile-api ", log.Lshortfile)
	log.Fatal(apiServer.Start())
}
