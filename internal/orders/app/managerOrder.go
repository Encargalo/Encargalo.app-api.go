package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type AppManagerOrder interface {
	Register(ctx context.Context, shopID uuid.UUID, conn *websocket.Conn) error
	Unregister(shopID uuid.UUID)
	BroadcastOrder(order models.Order) error
}
type appManagerOrder struct {
	mu          sync.RWMutex
	connections map[uuid.UUID]*websocket.Conn
	svc         ports.OrdersApp
}

func NewManagerOrderApp(svc ports.OrdersApp) AppManagerOrder {
	return &appManagerOrder{
		connections: make(map[uuid.UUID]*websocket.Conn),
		svc:         svc,
	}
}

func (s *appManagerOrder) Register(ctx context.Context, shopID uuid.UUID, conn *websocket.Conn) error {
	s.mu.Lock()
	s.connections[shopID] = conn
	s.mu.Unlock()

	orders, err := s.svc.SearchOrderByBatchIsOpen(ctx, shopID)
	if err != nil {
		if errors.Is(err, errcustom.ErrNotFound) {
			slog.Info("No hay órdenes abiertas para el local:", "shopID", shopID)
		}
		return err
	}

	for _, v := range orders {
		if err := s.BroadcastOrder(v); err != nil {
			slog.Error("Error al enviar orden abierta al local:", "shopID", shopID, "error", err)
		}
	}
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

	data, _ := json.Marshal(order)

	for id, conn := range s.connections {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			fmt.Printf("Error enviando mensaje a %s: %v\n", id, err)
			continue
		}
		fmt.Println("Orden enviada al local:", id)
	}

	defer s.mu.RUnlock()
	return nil
}
