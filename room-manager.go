package api

import (
	"context"
	"sync"
	"time"
)

// RoomManager is room manager.
type RoomManager struct {
	rooms *sync.Map
}

// Store stores the room.
func (m *RoomManager) Store(room *Room) {
	room.updated = time.Now()
	key := room.ApplicationName + room.Version
	roomMap, ok := m.rooms.Load(key)
	if ok {
		rooms, ok := roomMap.(*sync.Map)
		if ok {
			rooms.Store(room.RoomID, room)
			return
		}
	}

	rooms := &sync.Map{}
	rooms.Store(room.RoomID, room)
	m.rooms.Store(key, rooms)
}

// FindRoom returns the room.
func (m *RoomManager) FindRoom(roomID int) (room *Room) {
	m.rooms.Range(func(_, value interface{}) bool {
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
	m.rooms.Range(func(_, value interface{}) bool {
		rooms, ok := value.(*sync.Map)
		if !ok {
			return true
		}

		rooms.Delete(roomID)
		return true
	})
}

// Search returns returns all rooms with matching application name and version.
func (m *RoomManager) Search(name, version string) (rooms []*Room) {
	v, ok := m.rooms.Load(name + version)
	if !ok {
		return
	}

	r, ok := v.(*sync.Map)
	if !ok {
		return
	}

	r.Range(func(_, value interface{}) bool {
		room, ok := value.(*Room)
		if !ok {
			return true
		}
		if room.ApplicationName == name && room.Version == version {
			rooms = append(rooms, room)
		}
		return true
	})

	return
}

// DeleteDeadRoomAtPeriodic removes dead rooms at regular intervals
func (m *RoomManager) DeleteDeadRoomAtPeriodic(ctx context.Context, duration time.Duration) {
	ticker := time.NewTicker(duration)
	for {
		select {
		case <-ticker.C:
			var deadRoomIDs []int
			m.rooms.Range(func(_, value interface{}) bool {
				rooms, ok := value.(*sync.Map)
				if !ok {
					return true
				}
				rooms.Range(func(_, value interface{}) bool {
					room, ok := value.(*Room)
					if !ok {
						return true
					}
					if time.Now().Sub(room.updated) > duration {
						deadRoomIDs = append(deadRoomIDs, room.RoomID)
					}
					return true
				})
				return true
			})
			for _, id := range deadRoomIDs {
				m.Delete(id)
			}
		case <-ctx.Done():
			return
		}
	}
}
