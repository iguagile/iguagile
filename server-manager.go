package api

import (
	"sync"
)

// ServerManager is room server manager.
type ServerManager struct {
	servers *sync.Map
}

// Store stores the server.
func (m *ServerManager) Store(server *Server) {
	m.servers.Store(server.ServerID, server)
}

// Delete deletes the server.
func (m *ServerManager) Delete(serverID int) {
	m.servers.Delete(serverID)
}

// LoadServer returns the server.
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

// LowLoadServer returns the server with the lowest load.
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

// LoadServers returns all servers.
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
