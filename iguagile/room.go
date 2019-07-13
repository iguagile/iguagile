package iguagile

import (
	"encoding/binary"
	"log"
	"math"
	"os"

	"github.com/iguagile/iguagile-engine/data"
	"github.com/iguagile/iguagile-engine/id"
)

// Room maintains the set of active clients and broadcasts messages to the
// clients.
type Room struct {
	id        int
	clients   map[Client]bool
	buffer    map[*[]byte]Client
	objects   map[int]*GameObject
	generator *id.Generator
	log       *log.Logger
	host      Client
}

// NewRoom is Room constructed.
func NewRoom(serverID int, store Store) *Room {
	roomID, err := store.GenerateRoomID(serverID)
	if err != nil {
		log.Fatal(err)
	}

	gen, err := id.NewGenerator(math.MaxInt16)
	if err != nil {
		log.Fatal(err)
	}

	return &Room{
		id:        roomID,
		clients:   make(map[Client]bool),
		buffer:    make(map[*[]byte]Client),
		generator: gen,
		log:       log.New(os.Stdout, "iguagile-engine ", log.Lshortfile),
	}
}

// RPC target
const (
	AllClients = iota
	OtherClients
	AllClientsBuffered
	OtherClientsBuffered
	Host
	Server
)

// Message type
const (
	newConnection = iota
	exitConnection
	instantiate
	destroy
	requestObjectControlAuthority
	transferObjectControlAuthority
	migrateHost
)

const (
	// Maximum message size allowed from peer.
	maxMessageSize = 1<<16 - 1
)

// Register requests from the clients.
func (r *Room) Register(client Client) {
	go client.Run()
	message := append(client.GetIDByte(), newConnection)
	r.SendToOtherClients(message, client)
	r.clients[client] = true
	for msg := range r.buffer {
		client.Send(*msg)
	}
	r.buffer[&message] = client

	if len(r.clients) == 1 {
		r.host = client
		message := append(client.GetIDByte(), migrateHost)
		client.Send(message)
	}
}

// Unregister requests from clients.
func (r *Room) Unregister(client Client) {
	cid := client.GetID()
	for message, c := range r.buffer {
		if c == client {
			delete(r.buffer, message)
		}
	}
	r.generator.Free(cid)
	delete(r.clients, client)

	if client == r.host && len(r.clients) > 0 {
		for c := range r.clients {
			r.host = c
			message := append(c.GetIDByte(), migrateHost)
			c.Send(message)
			break
		}
	}
}

// Receive is receive inbound messages from the clients.
func (r *Room) Receive(sender Client, receivedData []byte) {
	rowData, err := data.NewInBoundData(receivedData)
	if err != nil {
		r.log.Println(err)
	}

	message := append(append(sender.GetIDByte(), rowData.MessageType), rowData.Payload...)
	if len(message) >= 1<<16 {
		r.log.Println("too long message")
		return
	}

	switch rowData.Target {
	case OtherClients:
		r.SendToOtherClients(message, sender)
	case AllClients:
		r.SendToAllClients(message)
	case OtherClientsBuffered:
		r.SendToOtherClients(message, sender)
		r.buffer[&message] = sender
	case AllClientsBuffered:
		r.SendToAllClients(message)
		r.buffer[&message] = sender
	case Host:
		r.host.Send(message)
	case Server:
		r.ReceiveRPC(sender, &rowData)
	default:
		r.log.Println(receivedData)
	}
}

// ReceiveRPC receives rpc to server
func (r *Room) ReceiveRPC(sender Client, binaryData *data.BinaryData) {
	switch binaryData.MessageType {
	case instantiate:
		r.InstantiateObject(sender, binaryData.Payload)
	case destroy:
		r.DestroyObject(sender, binaryData.Payload)
	case requestObjectControlAuthority:
		r.RequestObjectControlAuthority(sender, binaryData.Payload)
	case transferObjectControlAuthority:
		r.TransferObjectControlAuthority(sender, binaryData.Payload)
	case migrateHost:
		r.MigrateHost(sender, binaryData.Payload)
	default:
		r.log.Println(binaryData)
	}
}
q
// InstantiateObject instantiates the game object
func (r *Room) InstantiateObject(sender Client, idByte []byte) {
	objID := int(binary.LittleEndian.Uint32(idByte))
	if _, ok := r.objects[objID]; ok {
		return
	}

	r.objects[objID] = &GameObject{
		owner: sender,
		id:    objID,
	}

	message := append(append(sender.GetIDByte(), instantiate), idByte...)
	r.SendToAllClients(message)
}

// DestroyObject destroys the game object
func (r *Room) DestroyObject(sender Client, idByte []byte) {
	objID := int(binary.LittleEndian.Uint32(idByte))
	obj, ok := r.objects[objID]
	if !ok {
		return
	}

	if obj.owner != sender {
		return
	}

	delete(r.objects, objID)

	message := append(append(sender.GetIDByte(), destroy), idByte...)
	r.SendToAllClients(message)
}

// RequestObjectControlAuthority requests control authority of the object to the owner of the object
func (r *Room) RequestObjectControlAuthority(sender Client, idByte []byte) {
	objID := int(binary.LittleEndian.Uint32(idByte))
	obj, ok := r.objects[objID]
	if !ok {
		return
	}

	message := append(append(sender.GetIDByte(), requestObjectControlAuthority), idByte...)
	obj.owner.Send(message)
}

// TransferObjectControlAuthority transfers control authority of the object
func (r *Room) TransferObjectControlAuthority(sender Client, payload []byte) {
	objIDByte := payload[:4]
	objID := int(binary.LittleEndian.Uint32(objIDByte))

	clientIDByte := payload[4:8]
	clientID := int(binary.LittleEndian.Uint32(clientIDByte))

	obj, ok := r.objects[objID]
	if !ok {
		return
	}

	if obj.owner != sender {
		return
	}

	message := append(append(sender.GetIDByte(), transferObjectControlAuthority), objIDByte...)
	for client := range r.clients {
		if client.GetID() == clientID {
			client.Send(message)
		}
	}
}

// MigrateHost migrates host to the client
func (r *Room) MigrateHost(sender Client, idByte []byte) {
	clientID := int(binary.LittleEndian.Uint32(idByte))

	for client := range r.clients {
		if client.GetID() == clientID {
			message := append(client.GetIDByte(), migrateHost)
			client.Send(message)
			break
		}
	}
}

// SendToAllClients sends outbound message to all registered clients.
func (r *Room) SendToAllClients(message []byte) {
	for client := range r.clients {
		client.Send(message)
	}
}

// SendToOtherClients sends outbound message to other registered clients.
func (r *Room) SendToOtherClients(message []byte, sender Client) {
	for client := range r.clients {
		if client != sender {
			client.Send(message)
		}
	}
}

// CloseConnection closes the connection and unregisters the client.
func (r *Room) CloseConnection(client Client) {
	message := append(client.GetIDByte(), exitConnection)
	r.SendToOtherClients(message, client)
	r.Unregister(client)
	if err := client.Close(); err != nil && err.Error() != "use of closed network connection" {
		r.log.Println(err)
	}
}

// Close closes all client connections
func (r *Room) Close() error {
	for client := range r.clients {
		if err := client.Close(); err != nil {
			r.log.Println(err)
		}
	}

	return nil
}
