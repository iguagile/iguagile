package main

import (
	"github.com/labstack/echo"
)

type Server struct {
	Host string `json:"server"`
	Port int    `json:"port"`
}

type Room struct {
	RoomID          int    `json:"room_id"`
	RequirePassword bool   `json:"require_password"`
	MaxUser         int    `json:"max_user"`
	ConnectedUser   int    `json:"connected_user"`
	Server          Server `json:"server"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Result  interface{} `json:"result"`
	Error   string      `json:"error"`
}

const iguagileAPIVersion = "v1"

func main() {
	e := echo.New()
	g := e.Group("/api/v1")
	g.Add(echo.POST, "/create", roomCreateHandler)
	g.Add(echo.GET, "/search", roomListHandler)
	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Add("X-IGUAGILE-API", iguagileAPIVersion)
			return next(c)
		}
	})
	e.Logger.Fatal(e.Start("localhost:1323"))
}

func roomCreateHandler(c echo.Context) error {
	res := APIResponse{
		Success: true,
		Result: Room{
			RoomID:  1,
			MaxUser: 10,
			Server: Server{
				Host: "localhost",
				Port: 4000,
			},
		},
	}
	return c.JSON(201, res)
}

func roomListHandler(c echo.Context) error {
	var rooms []Room
	rooms = append(rooms, Room{
		RoomID:          1,
		RequirePassword: false,
		Server: Server{
			Host: "localhost",
			Port: 4000,
		},
		MaxUser:       10,
		ConnectedUser: 1,
	})
	res := APIResponse{
		Success: true,

		Result: rooms,
	}

	return c.JSON(200, res)
}
