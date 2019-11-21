package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/gomodule/redigo/redis"
	pb "github.com/iguagile/iguagile-room-proto/room"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"google.golang.org/grpc"
)

type Server struct {
	Host     string `json:"server"`
	Port     int    `json:"port"`
	ServerID int
	Load     int
	APIPort  int
	Token    []byte
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

func (m *ServerManager) LoadServer(serverID int) (server *Server) {
	m.servers.Range(func(key, value interface{}) bool {
		s, ok := value.(*Server)
		if ok || s.ServerID == serverID {
			server = s
			return false
		}
		return true
	})

	return
}

func (m *ServerManager) LowLoadServer() (server *Server) {
	m.servers.Range(func(key, value interface{}) bool {
		s, ok := value.(*Server)
		if ok || server == nil || server.Load > s.Load {
			server = s
		}
		return true
	})

	return
}

func (m *ServerManager) LoadServers() (servers []*Server) {
	m.servers.Range(func(key, value interface{}) bool {
		server, ok := value.(*Server)
		if ok {
			servers = append(servers, server)
		}
		return true
	})

	return
}

type RoomManager struct {
	rooms *sync.Map
}

func (m *RoomManager) LoadRooms(applicationName, version string) (results []*pb.Room) {
	roomMap, ok := m.rooms.Load(applicationName + version)
	if !ok {
		return
	}

	rooms, ok := roomMap.(*sync.Map)
	if !ok {
		return
	}

	rooms.Range(func(key, value interface{}) bool {
		room, ok := value.(*pb.Room)
		if ok {
			results = append(results, room)
		}
		return true
	})

	return
}

func (m *RoomManager) Store(room *Room) {
	m.rooms.Store(room.RoomID, room)
	key := room.ApplicationName + room.Version
	roomMap, ok := m.rooms.Load(key)
	if !ok {
		rooms := &sync.Map{}
		rooms.Store(room.RoomID, room)
		m.rooms.Store(key, rooms)
	} else {
		rooms, ok := roomMap.(*sync.Map)
		if ok {
			rooms.Store(room.RoomID, room)
		}
	}
}

func (m *RoomManager) LoadRoom(roomID int) (room *Room) {
	m.rooms.Range(func(key, value interface{}) bool {
		rooms, ok := value.(*sync.Map)
		if !ok {
			return true
		}

		r, ok := rooms.Load(roomID)
		if !ok {
			return true
		}

		room, ok = r.(*Room)
		return !ok
	})

	return
}

func (m *RoomManager) Delete(roomID int) {
	m.rooms.Range(func(key, value interface{}) bool {
		rooms, ok := value.(*sync.Map)
		if !ok {
			return true
		}

		rooms.Delete(roomID)
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

var (
	serverManager = &ServerManager{servers: &sync.Map{}}
	roomManager   = &RoomManager{rooms: &sync.Map{}}
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

	go subscribe(psc)

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

func subscribe(psc redis.PubSubConn) {
	for {
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

					if server := serverManager.LoadServer(int(roomProto.Server.ServerId)); server != nil {
						if room := roomManager.LoadRoom(int(roomProto.RoomId)); room != nil {
							server.Load += int(roomProto.ConnectedUser*roomProto.ConnectedUser) - room.ConnectedUser*room.ConnectedUser
						}
					}
				case unregisterRoomMessage:
					roomProto := &pb.Room{}
					if err := proto.Unmarshal(v.Data[1:], roomProto); err != nil {
						log.Println(err)
						break
					}

					if server := serverManager.LoadServer(int(roomProto.Server.ServerId)); server != nil {
						server.Load -= int(roomProto.ConnectedUser * roomProto.ConnectedUser)
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
			log.Println(v)
		}
	}
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

	server := serverManager.LowLoadServer()
	if server == nil {
		return fmt.Errorf("server not exists")
	}

	grpcHost := fmt.Sprintf("%v:%v", server.Host, server.APIPort)
	grpcConn, err := grpc.Dial(grpcHost, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer func() { _ = grpcConn.Close() }()
	grpcClient := pb.NewRoomServiceClient(grpcConn)
	grpcRequest := &pb.CreateRoomRequest{
		ApplicationName: request.ApplicationName,
		Version:         request.Version,
		Password:        request.Password,
		MaxUser:         int32(request.MaxUser),
	}
	grpcResponse, err := grpcClient.CreateRoom(context.Background(), grpcRequest)
	if err != nil {
		return err
	}

	room := &Room{
		RoomID:          int(grpcResponse.Room.RoomId),
		MaxUser:         int(grpcResponse.Room.MaxUser),
		RequirePassword: grpcResponse.Room.RequirePassword,
		Server: Server{
			Host: server.Host,
			Port: server.Port,
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

	rooms := roomManager.Search(name, version)
	res := APIResponse{
		Success: true,
		Result:  rooms,
	}

	return c.JSON(200, res)
}
