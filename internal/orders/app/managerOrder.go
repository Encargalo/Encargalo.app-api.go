package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

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
	connections map[uuid.UUID][]*websocket.Conn
	svc         ports.OrdersApp
}

func NewManagerOrderApp(svc ports.OrdersApp) AppManagerOrder {
	return &appManagerOrder{
		connections: make(map[uuid.UUID][]*websocket.Conn),
		svc:         svc,
	}
}

func (s *appManagerOrder) Register(ctx context.Context, shopID uuid.UUID, conn *websocket.Conn) error {
	s.mu.Lock()
	s.connections[shopID] = append(s.connections[shopID], conn)
	s.mu.Unlock()

	orders, err := s.svc.SearchOrderByBatchIsOpen(ctx, shopID)
	if err != nil {
		if !errors.Is(err, errcustom.ErrNotFound) {
			slog.Error("Error al buscar órdenes abiertas para el local:", "shopID", shopID, "error", err)
			return err
		}
		slog.Info("No hay órdenes abiertas para el local:", "shopID", shopID)
	}

	for _, v := range orders {
		if err := s.BroadcastOrder(v); err != nil {
			slog.Error("Error al enviar orden abierta al local:", "shopID", shopID, "error", err)
		}
	}

	go s.ping(conn)

	return nil
}

func (s *appManagerOrder) Unregister(shopID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.connections, shopID)
	slog.Info("Conexión cerrada para el local:", "shopID", shopID)
}

func (s *appManagerOrder) BroadcastOrder(order models.Order) error {
	data, err := json.Marshal(order)
	if err != nil {
		return err
	}

	s.mu.RLock()
	conns, exists := s.connections[order.ShopID]
	s.mu.RUnlock()

	if !exists {
		slog.Warn("No existen conexiones activas para el local", "shopID", order.ShopID)
		return nil
	}

	for _, conn := range conns {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			slog.Error("Error enviando mensaje", "shopID", order.ShopID, "error", err)
			continue
		}
	}

	slog.Info("Orden enviada a todas las conexiones", "shopID", order.ShopID, "total", len(conns))
	return nil
}

func (s *appManagerOrder) ping(conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for range ticker.C {
		if err := conn.WriteControl(
			websocket.PingMessage,
			[]byte("ping"),
			time.Now().Add(5*time.Second),
		); err != nil {
			slog.Error("Error enviando ping:", "error", err)
			conn.Close()
			return
		}
	}
}
