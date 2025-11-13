package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"Encargalo.app-api.go/internal/orders/domain/models"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type AppManagerOrder interface {
	Register(shopID uuid.UUID, conn *websocket.Conn) error
	Unregister(shopID uuid.UUID)
	BroadcastOrder(order models.Order) error
}
type appManagerOrder struct {
	mu          sync.RWMutex
	connections map[uuid.UUID]*websocket.Conn
}

func NewManagerOrderApp() AppManagerOrder {
	return &appManagerOrder{
		connections: make(map[uuid.UUID]*websocket.Conn),
	}
}

func (s *appManagerOrder) Register(shopID uuid.UUID, conn *websocket.Conn) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.connections[shopID] = conn
	slog.Info("Conectado local:", "shopID", shopID)
	return nil
}

func (s *appManagerOrder) Unregister(shopID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.connections, shopID)
	fmt.Println("Desconectado local:", shopID)
}

func (s *appManagerOrder) BroadcastOrder(order models.Order) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, _ := json.Marshal(order)

	for id, conn := range s.connections {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			fmt.Printf("Error enviando mensaje a %s: %v\n", id, err)
			continue
		}
		fmt.Println("Orden enviada al local:", id)
	}

	return nil
}
