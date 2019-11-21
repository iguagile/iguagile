package api

import (
	"sync"

	"github.com/gomodule/redigo/redis"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

// Server is room server information.
type Server struct {
	Host     string `json:"server"`
	Port     int    `json:"port"`
	ServerID int
	Load     int
	APIPort  int
	Token    []byte
}

// Room is room information.
type Room struct {
	RoomID          int    `json:"room_id"`
	RequirePassword bool   `json:"require_password"`
	MaxUser         int    `json:"max_user"`
	ConnectedUser   int    `json:"connected_user"`
	Server          Server `json:"server"`
	ApplicationName string
	Version         string
}

// RoomAPIResponse is api response.
type RoomAPIResponse struct {
	Success bool        `json:"success"`
	Result  interface{} `json:"result"`
	Error   string      `json:"error"`
}

// CreateRoomRequest is api request.
type CreateRoomRequest struct {
	ApplicationName string `json:"application_name"`
	Version         string `json:"version"`
	Password        string `json:"password"`
	MaxUser         int    `json:"max_user"`
}

const iguagileAPIVersion = "v1"

var (
	serverManager = &ServerManager{servers: &sync.Map{}}
	roomManager   = &RoomManager{rooms: &sync.Map{}}
)

var (
	// Address is room api server address.
	Address   = ":80"

	// BaseUri is base uri of room api.
	BaseUri   = "/api/v1"

	// RedisHost is redis address.
	RedisHost = ":6379"

	// MaxUser is max value of room capacity.
	MaxUser   = 70
)

// Start starts an room api server.
func Start() error {
	redisConn, err := redis.Dial("tcp", RedisHost)
	if err != nil {
		return err
	}

	psc := redis.PubSubConn{Conn: redisConn}
	if err := psc.Subscribe(channelServer, channelRoom); err != nil {
		return err
	}

	go subscribe(psc)

	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	g := e.Group(BaseUri)
	g.Add(echo.POST, "/rooms", roomCreateHandler)
	g.Add(echo.GET, "/rooms", roomListHandler)
	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Add("X-IGUAGILE-API", iguagileAPIVersion)
			return next(c)
		}
	})

	return e.Start(Address)
}
