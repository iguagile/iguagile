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

func (s *RoomAPIServer) subscribe(psc redis.PubSubConn) {
	for {
		switch v := psc.Receive().(type) {
		case redis.Message:
			if len(v.Data) <= 1 {
				s.Logger.Printf("invalid message %v\n", v)
				break
			}
			switch v.Channel {
			case channelRoom:
				room := &pb.Room{}
				if err := proto.Unmarshal(v.Data[1:], room); err != nil {
					s.Logger.Println(err)
					break
				}
				switch v.Data[0] {
				case registerRoomMessage:
					s.registerRoom(room)
				case unregisterRoomMessage:
					s.unregisterRoom(room)
				default:
					s.Logger.Printf("invalid message type %v\n", v)
				}
			case channelServer:
				server := &pb.Server{}
				if err := proto.Unmarshal(v.Data[1:], server); err != nil {
					s.Logger.Println(err)
					break
				}
				switch v.Data[0] {
				case registerServerMessage:
					s.registerServer(server)
				case unregisterServerMessage:
					s.unregisterServer(server)
				default:
					s.Logger.Printf("invalid message type %v\n", v)
				}
			default:
				s.Logger.Printf("invalid channel%v\n", v)
			}
		case redis.Subscription:
			s.Logger.Printf("Subscribe %v %v %v\n", v.Channel, v.Kind, v.Count)
		case error:
			s.Logger.Println(v)
		}
	}
}

func (s *RoomAPIServer) registerRoom(room *pb.Room) {
	s.roomManager.Store(&Room{
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

	if server := s.serverManager.LoadServer(int(room.Server.ServerId)); server != nil {
		if r := s.roomManager.FindRoom(int(room.RoomId)); room != nil {
			server.Load += int(room.ConnectedUser*room.ConnectedUser) - r.MaxUser*r.MaxUser
		}
	}
}

func (s *RoomAPIServer) unregisterRoom(room *pb.Room) {
	if server := s.serverManager.LoadServer(int(room.Server.ServerId)); server != nil {
		server.Load -= int(room.ConnectedUser * room.ConnectedUser)
	}

	s.roomManager.Delete(int(room.RoomId))
}

func (s *RoomAPIServer) registerServer(server *pb.Server) {
	s.serverManager.Store(&Server{
		Host:     server.Host,
		Port:     int(server.Port),
		ServerID: int(server.ServerId),
	})
}

func (s *RoomAPIServer) unregisterServer(server *pb.Server) {
	s.serverManager.Delete(int(server.ServerId))
}
