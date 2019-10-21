package main

import (
	"github.com/labstack/echo"
)

type Server struct {
	Host string `json:"server"`
	Port int    `json:"port"`
}

type RoomCreateResponse struct {
	RoomID  int    `json:"room_id"`
	MaxUser int    `json:"max_user"`
	Server  Server `json:"server"`
}

type RoomSearchResponse struct {
	RoomID          int    `json:"room_id"`
	RequirePassword bool   `json:"require_password"`
	Server          Server `json:"server"`
	MaxUser         int    `json:"max_user"`
	ConnectedUser   int    `json:"connected_user"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Result  interface{} `json:"result"`
	Error   string      `json:"error"`
}

const iguagileAPIVersion = "v1"

func main() {
	e := echo.New()
	g := e.Group("/v1/api")
	g.Add(echo.POST, "/create", roomCreateHandler)
	g.Add(echo.GET, "/search", roomListHandler)
	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Add("X-IGUAGILE-API", iguagileAPIVersion)
			return next(c)
		}
	})
	//e.Host("localhost")
	e.Logger.Fatal(e.Start("localhost:1323"))
}

func roomCreateHandler(c echo.Context) error {
	res := APIResponse{
		Success: true,
		Result: RoomCreateResponse{
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
	var rooms []RoomSearchResponse
	rooms = append(rooms, RoomSearchResponse{
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
