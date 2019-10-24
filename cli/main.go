package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math"
	"sync"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/minami14/idgo"
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
	ApplicationName string
	Version         string
}

type APIResponse struct {
	Success bool        `json:"success"`
	Result  interface{} `json:"result"`
	Error   string      `json:"error"`
}

type CreateRoomRequest struct {
	ApplicationName string `json:"application_name"`
	Version         string `json:"version"`
	Password        string `json:"password"`
	MaxUser         int    `json:"max_user"`
}

type RoomManager struct {
	rooms *sync.Map
}

func (m *RoomManager) Store(room *Room) {
	m.rooms.Store(room.RoomID, room)
}

func (m *RoomManager) Search(name, version string) []*Room {
	rooms := make([]*Room, 0)
	m.rooms.Range(func(key, value interface{}) bool {
		room := value.(*Room)
		if room.ApplicationName == name && room.Version == version {
			rooms = append(rooms, room)
		}
		return true
	})

	return rooms
}

var roomManager = &RoomManager{
	rooms: &sync.Map{},
}

var generator *idgo.IDGenerator

const iguagileAPIVersion = "v1"

const maxUser = 70

func main() {
	store, err := idgo.NewLocalStore(math.MaxInt16)
	if err != nil {
		log.Fatal(err)
	}

	generator, err = idgo.NewIDGenerator(store)
	if err != nil {
		log.Fatal(err)
	}

	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
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
	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}

	var request CreateRoomRequest
	if err := json.Unmarshal(body, &request); err != nil {
		return err
	}
	log.Printf("create %v\n", request)

	if request.MaxUser > maxUser {
		res := APIResponse{
			Success: false,
			Error:   "MaxUser exceeds the maximum value",
		}
		return c.JSON(400, res)
	}

	id, err := generator.Generate()
	if err != nil {
		return err
	}

	room := &Room{
		RoomID:          id,
		MaxUser:         request.MaxUser,
		RequirePassword: request.Password != "",
		Server: Server{
			Host: "localhost",
			Port: 4000,
		},
		ApplicationName: request.ApplicationName,
		Version:         request.Version,
	}

	roomManager.Store(room)

	res := APIResponse{
		Success: true,
		Result:  room,
	}
	return c.JSON(201, res)
}

func roomListHandler(c echo.Context) error {
	name := c.QueryParam("name")
	version := c.QueryParam("version")
	log.Printf("search %s %s\n", name, version)

	rooms := roomManager.Search(name, version)
	res := APIResponse{
		Success: true,
		Result:  rooms,
	}

	return c.JSON(200, res)
}
