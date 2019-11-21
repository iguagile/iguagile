package api

import (
	"github.com/golang/protobuf/proto"
	"github.com/gomodule/redigo/redis"
	pb "github.com/iguagile/iguagile-room-proto/room"
)

const (
	channelServer = "channel_servers"
	channelRoom   = "channel_rooms"
)

const (
	registerServerMessage = iota
	unregisterServerMessage
	registerRoomMessage
	unregisterRoomMessage
)

func subscribe(psc redis.PubSubConn) {
	for {
		switch v := psc.Receive().(type) {
		case redis.Message:
			if len(v.Data) <= 1 {
				Logger.Printf("invalid message %v\n", v)
				break
			}
			switch v.Channel {
			case channelRoom:
				room := &pb.Room{}
				if err := proto.Unmarshal(v.Data[1:], room); err != nil {
					Logger.Println(err)
					break
				}
				switch v.Data[0] {
				case registerRoomMessage:
					registerRoom(room)
				case unregisterRoomMessage:
					unregisterRoom(room)
				default:
					Logger.Printf("invalid message type %v\n", v)
				}
			case channelServer:
				server := &pb.Server{}
				if err := proto.Unmarshal(v.Data[1:], server); err != nil {
					Logger.Println(err)
					break
				}
				switch v.Data[0] {
				case registerServerMessage:
					registerServer(server)
				case unregisterServerMessage:
					unregisterServer(server)
				default:
					Logger.Printf("invalid message type %v\n", v)
				}
			default:
				Logger.Printf("invalid channel%v\n", v)
			}
		case redis.Subscription:
			Logger.Printf("Subscribe %v %v %v\n", v.Channel, v.Kind, v.Count)
		case error:
			Logger.Println(v)
		}
	}
}

func registerRoom(room *pb.Room) {
	roomManager.Store(&Room{
		RoomID:          int(room.RoomId),
		RequirePassword: room.RequirePassword,
		MaxUser:         int(room.MaxUser),
		ConnectedUser:   int(room.ConnectedUser),
		Server: Server{
			Host:     room.Server.Host,
			Port:     int(room.Server.Port),
			ServerID: int(room.Server.ServerId),
		},
		ApplicationName: room.ApplicationName,
		Version:         room.Version,
	})

	if server := serverManager.LoadServer(int(room.Server.ServerId)); server != nil {
		if room := roomManager.LoadRoom(int(room.RoomId)); room != nil {
			server.Load += int(room.ConnectedUser*room.ConnectedUser) - room.MaxUser*room.MaxUser
		}
	}
}

func unregisterRoom(room *pb.Room) {
	if server := serverManager.LoadServer(int(room.Server.ServerId)); server != nil {
		server.Load -= int(room.ConnectedUser * room.ConnectedUser)
	}

	roomManager.Delete(int(room.RoomId))
}

func registerServer(server *pb.Server) {
	serverManager.Store(&Server{
		Host:     server.Host,
		Port:     int(server.Port),
		ServerID: int(server.ServerId),
	})
}

func unregisterServer(server *pb.Server) {
	serverManager.Delete(int(server.ServerId))
}
