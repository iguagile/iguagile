package main

import (
	"github.com/labstack/echo"
)

func main() {
	e := echo.New()
	g := e.Group("/v1/api")
	g.Add(echo.POST, "/create", roomCreateHandler)
	g.Add(echo.GET, "/search", roomListHandler)
	e.Logger.Fatal(e.Start(":1323"))
}

func roomCreateHandler(c echo.Context) error {

	//language=JSON
	_ := `
{
	"room_id": "number",
  	"password": "string",
  	"version": "string",
  	"max_user": "number",
	"server": {
		"host": "string",
		"port": "number"
	},
	"join_token": "string"
 
}
	`
	return nil
}

func roomListHandler(c echo.Context) error {
	//language=JSON
	_ := `
[
	{
        "room_id": "number",
        "require_password": "bool",
        "max_user": "number",
        "connected_user": "number"
    }
]
	`

	return nil
}
