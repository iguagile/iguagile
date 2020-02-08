package api

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
	pb "github.com/iguagile/iguagile-room-proto/room"
	"github.com/labstack/echo"
	"google.golang.org/grpc"
)

var (
	exceedResponse = RoomAPIResponse{
		Success: false,
		Error:   "MaxUser exceeds the maximum value",
	}

	errNoServer = fmt.Errorf("server not exists")
)

func (s *RoomAPIServer) roomCreateHandler(c echo.Context) error {
	request := &CreateRoomRequest{}
	if err := c.Bind(request); err != nil {
		return err
	}

	if request.MaxUser > s.MaxUser {
		return c.JSON(400, exceedResponse)
	}

	server := s.serverManager.PickupLowLoadServer()
	if server == nil {
		return errNoServer
	}

	grpcHost := fmt.Sprintf("%v:%v", server.Host, server.APIPort)
	grpcConn, err := grpc.Dial(grpcHost, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer func() { _ = grpcConn.Close() }()

	roomToken := uuid.New()
	grpcClient := pb.NewRoomServiceClient(grpcConn)
	grpcRequest := &pb.CreateRoomRequest{
		ApplicationName: request.ApplicationName,
		Version:         request.Version,
		Password:        request.Password,
		MaxUser:         int32(request.MaxUser),
		ServerToken:     server.Token,
		RoomToken:       roomToken[:],
		Information:     request.Information,
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
		Token:           base64.StdEncoding.EncodeToString(roomToken[:]),
		Information:     request.Information,
	}

	res := RoomAPIResponse{
		Success: true,
		Result:  room,
	}

	defer func() {
		room.Token = ""
		s.roomManager.Store(room)
	}()

	return c.JSON(201, res)
}

func (s *RoomAPIServer) roomListHandler(c echo.Context) error {
	name := c.QueryParam("name")
	version := c.QueryParam("version")

	rooms := s.roomManager.Search(name, version)
	res := RoomAPIResponse{
		Success: true,
		Result:  rooms,
	}

	return c.JSON(200, res)
}
