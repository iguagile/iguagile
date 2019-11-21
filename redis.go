package api

import (
	"log"

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

					if server := serverManager.LoadServer(int(roomProto.Server.ServerId)); server != nil {
						if room := roomManager.LoadRoom(int(roomProto.RoomId)); room != nil {
							server.Load += int(roomProto.ConnectedUser*roomProto.ConnectedUser) - room.MaxUser*room.MaxUser
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
