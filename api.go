package api

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

// RoomAPIServer is room api server.
type RoomAPIServer struct {
	// Address is room api server address.
	Address string

	// BaseUri is base uri of room api.
	BaseUri string

	// RedisHost is redis address.
	RedisHost string

	// MaxUser is max value of room capacity.
	MaxUser int

	ServerDeadLine time.Duration
	RoomDeadLine   time.Duration
	Logger         *log.Logger

	serverManager *ServerManager
	roomManager   *RoomManager
}

const (
	defaultAddress        = ":80"
	defaultBaseUri        = "/api/v1"
	defaultRedisHost      = ":6379"
	defaultMaxUser        = 70
	defaultServerDeadline = time.Minute * 5
	defaultRoomDeadline   = time.Minute * 5
)

// NewRoomAPIServer is an instance of RoomAPIServer.
func NewRoomAPIServer() *RoomAPIServer {
	return &RoomAPIServer{
		Address:        defaultAddress,
		BaseUri:        defaultBaseUri,
		RedisHost:      defaultRedisHost,
		MaxUser:        defaultMaxUser,
		ServerDeadLine: defaultServerDeadline,
		RoomDeadLine:   defaultRoomDeadline,
		Logger:         log.New(os.Stdout, "iguagile-room-api ", log.Lshortfile),
		serverManager:  &ServerManager{servers: &sync.Map{}},
		roomManager:    &RoomManager{rooms: &sync.Map{}},
	}
}

// Server is room server information.
type Server struct {
	Host     string `json:"server"`
	Port     int    `json:"port"`
	ServerID int
	Load     int
	APIPort  int
	Token    []byte
	updated  time.Time
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
	updated         time.Time
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

// Start starts an room api server.
func (s *RoomAPIServer) Start() error {
	redisConn, err := redis.Dial("tcp", s.RedisHost)
	if err != nil {
		return err
	}

	psc := redis.PubSubConn{Conn: redisConn}
	if err := psc.Subscribe(channelServer, channelRoom); err != nil {
		return err
	}

	go s.subscribe(psc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go s.serverManager.StartRemoveDeadServer(ctx, s.ServerDeadLine)
	go s.roomManager.StartRemoveDeadRoom(ctx, s.RoomDeadLine)

	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	g := e.Group(s.BaseUri)
	g.Add(echo.POST, "/rooms", s.roomCreateHandler)
	g.Add(echo.GET, "/rooms", s.roomListHandler)
	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Add("X-IGUAGILE-API", iguagileAPIVersion)
			return next(c)
		}
	})

	return e.Start(s.Address)
}
