package api

import (
	"context"
	"sync"
	"time"
)

// ServerManager is room server manager.
type ServerManager struct {
	servers *sync.Map
}

// Store stores the server.
func (m *ServerManager) Store(server *Server) {
	server.updated = time.Now()
	m.servers.Store(server.ServerID, server)
}

// Delete deletes the server.
func (m *ServerManager) Delete(serverID int) {
	m.servers.Delete(serverID)
}

// LoadServer returns the server.
func (m *ServerManager) LoadServer(serverID int) (server *Server) {
	m.servers.Range(func(_, value interface{}) bool {
		s, ok := value.(*Server)
		if ok || s.ServerID == serverID {
			server = s
			return false
		}
		return true
	})

	return
}

// PickupLowLoadServer returns the server with the lowest load.
func (m *ServerManager) PickupLowLoadServer() (server *Server) {
	m.servers.Range(func(_, value interface{}) bool {
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
	m.servers.Range(func(_, value interface{}) bool {
		server, ok := value.(*Server)
		if ok {
			servers = append(servers, server)
		}
		return true
	})

	return
}

// DeleteUnhealthServerAtPeriodic removes dead servers at regular intervals
func (m *ServerManager) DeleteUnhealthServerAtPeriodic(ctx context.Context, duration time.Duration) {
	ticker := time.NewTicker(duration)
	for {
		select {
		case <-ticker.C:
			var deadServerIDs []int
			m.servers.Range(func(_, value interface{}) bool {
				server, ok := value.(*Server)
				if !ok {
					return true
				}
				if time.Now().Sub(server.updated) > duration {
					deadServerIDs = append(deadServerIDs, server.ServerID)
				}
				return true
			})
			for _, id := range deadServerIDs {
				m.Delete(id)
			}
		case <-ctx.Done():
			return
		}
	}
}
