package main

import (
	"log"
	"os"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/gomodule/redigo/redis"
	pb "github.com/iguagile/iguagile-room-proto/room"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/minami14/idgo"
)

type Server struct {
	Host     string `json:"server"`
	Port     int    `json:"port"`
	ServerID int
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

type ServerManager struct {
	servers *sync.Map
}

func (m *ServerManager) Store(server *Server) {
	m.servers.Store(server.ServerID, server)
}

func (m *ServerManager) Delete(serverID int) {
	m.servers.Delete(serverID)
}

func (m *ServerManager) LoadServers() []*Server {
	var servers []*Server
	m.servers.Range(func(key, value interface{}) bool {
		switch server := value.(type) {
		case *Server:
			servers = append(servers, server)
		}
		return true
	})

	return servers
}

type RoomManager struct {
	rooms *sync.Map
}

func (m *RoomManager) LoadRooms(applicationName, version string) []*pb.Room {
	rooms, ok := m.rooms.Load(applicationName + version)
	if !ok {
		return []*pb.Room{}
	}

	switch v := rooms.(type) {
	case *sync.Map:
		var rooms []*pb.Room
		v.Range(func(key, value interface{}) bool {
			switch v := value.(type) {
			case *pb.Room:
				rooms = append(rooms, v)
			}
			return true
		})
		return rooms
	default:
		return []*pb.Room{}
	}
}

func (m *RoomManager) Store(room *Room) {
	m.rooms.Store(room.RoomID, room)
	key := room.ApplicationName + room.Version
	rooms, ok := m.rooms.Load(key)
	if !ok {
		rooms := &sync.Map{}
		rooms.Store(room.RoomID, room)
		m.rooms.Store(key, rooms)
	} else {
		switch v := rooms.(type) {
		case *sync.Map:
			v.Store(room.RoomID, room)
		}
	}
}

func (m *RoomManager) Delete(roomID int) {
	m.rooms.Range(func(key, value interface{}) bool {
		switch rooms := value.(type) {
		case *sync.Map:
			rooms.Delete(roomID)
		}
		return true
	})
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

const (
	channelServer = "channel_servers"
	channelRoom   = "channel_rooms"
)

const (
	registerServerMessage = iota
	unregisterServerMessage
	registerRoomMessage
	unregisterRoomMessage
	updateRoomMessage
)

func main() {
	redisConn, err := redis.Dial("tcp", os.Getenv("REDIS_HOST"))
	if err != nil {
		log.Fatal(err)
	}

	psc := redis.PubSubConn{Conn: redisConn}
	if err := psc.Subscribe(channelServer, channelRoom); err != nil {
		log.Fatal(err)
	}

	serverManager := &ServerManager{servers: &sync.Map{}}
	roomManager := &RoomManager{rooms: &sync.Map{}}

	go func() {
		switch v := psc.Receive().(type) {
		case redis.Message:
			if len(v.Data) <= 1 {
				log.Printf("invalid message %v\n", v)
				break
			}
			switch v.Channel {
			case channelRoom:
				switch v.Data[0] {
				case registerRoomMessage:
					roomProto := &pb.Room{}
					if err := proto.Unmarshal(v.Data[1:], roomProto); err != nil {
						log.Println(err)
						break
					}

					roomManager.Store(&Room{
						RoomID:          int(roomProto.RoomId),
						RequirePassword: roomProto.RequirePassword,
						MaxUser:         int(roomProto.MaxUser),
						ConnectedUser:   int(roomProto.ConnectedUser),
						Server: Server{
							Host:     roomProto.Server.Host,
							Port:     int(roomProto.Server.Port),
							ServerID: int(roomProto.Server.ServerId),
						},
						ApplicationName: roomProto.ApplicationName,
						Version:         roomProto.Version,
					})
				case updateRoomMessage:
					roomProto := &pb.Room{}
					if err := proto.Unmarshal(v.Data[1:], roomProto); err != nil {
						log.Println(err)
						break
					}

					roomManager.Store(&Room{
						RoomID:          int(roomProto.RoomId),
						RequirePassword: roomProto.RequirePassword,
						MaxUser:         int(roomProto.MaxUser),
						ConnectedUser:   int(roomProto.ConnectedUser),
						Server: Server{
							Host:     roomProto.Server.Host,
							Port:     int(roomProto.Server.Port),
							ServerID: int(roomProto.Server.ServerId),
						},
						ApplicationName: roomProto.ApplicationName,
						Version:         roomProto.Version,
					})
				case unregisterRoomMessage:
					roomProto := &pb.Room{}
					if err := proto.Unmarshal(v.Data[1:], roomProto); err != nil {
						log.Println(err)
						break
					}

					roomManager.Delete(int(roomProto.RoomId))
				default:
					log.Printf("invalid message type %v\n", v)
				}
			case channelServer:
				switch v.Data[0] {
				case registerServerMessage:
					serverProto := &pb.Server{}
					if err := proto.Unmarshal(v.Data[1:], serverProto); err != nil {
						log.Println(err)
						break
					}

					serverManager.Store(&Server{
						Host:     serverProto.Host,
						Port:     int(serverProto.Port),
						ServerID: int(serverProto.ServerId),
					})
				case unregisterServerMessage:
					serverProto := &pb.Server{}
					if err := proto.Unmarshal(v.Data[1:], serverProto); err != nil {
						log.Println(err)
						break
					}

					serverManager.Delete(int(serverProto.ServerId))
				default:
					log.Printf("invalid message type %v\n", v)
				}
			default:
				log.Printf("invalid channel%v\n", v)
			}
		case redis.Subscription:
			log.Printf("Subscribe %v %v %v\n", v.Channel, v.Kind, v.Count)
		case error:
			log.Println(err)
		}
	}()

	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	g := e.Group("/api/v1")
	g.Add(echo.POST, "/rooms", roomCreateHandler)
	g.Add(echo.GET, "/rooms", roomListHandler)
	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Add("X-IGUAGILE-API", iguagileAPIVersion)
			return next(c)
		}
	})
	e.Logger.Fatal(e.Start("localhost:1323"))
}

func roomCreateHandler(c echo.Context) error {
	request := &CreateRoomRequest{}
	if err := c.Bind(request); err != nil {
		return err
	}

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
