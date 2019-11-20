package main

import (
	"github.com/golang/protobuf/proto"
	"github.com/gomodule/redigo/redis"
	pb "github.com/iguagile/iguagile-room-proto/room"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"log"
	"sync"
)



func run(apiAddress, redisHostName string) error {
	redisConn, err := redis.Dial("tcp", redisHostName)
	if err != nil {
		return err
	}

	psc := redis.PubSubConn{Conn: redisConn}
	if err := psc.Subscribe(channelServer, channelRoom); err != nil {
		return err
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
	return e.Start(apiAddress)
}

