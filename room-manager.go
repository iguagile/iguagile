package api

import (
	"sync"
)

// RoomManager is room manager.
type RoomManager struct {
	rooms *sync.Map
}

// Store stores the room.
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

// LoadRoom returns the room.
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

// Delete deletes the room.
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

// Search returns returns all rooms with matching application name and version.
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
