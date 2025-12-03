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
	Register(ctx context.Context, connectionID uuid.UUID, conn *websocket.Conn, isShop bool) error
	Unregister(connectionID uuid.UUID, isShop bool)
	BroadcastOrder(order models.Order) error
}
type appManagerOrder struct {
	mu                  sync.RWMutex
	shopConnections     map[uuid.UUID][]*websocket.Conn
	customerConnections map[uuid.UUID][]*websocket.Conn
	svc                 ports.OrdersApp
}

func NewManagerOrderApp(svc ports.OrdersApp) AppManagerOrder {
	return &appManagerOrder{
		shopConnections:     make(map[uuid.UUID][]*websocket.Conn),
		customerConnections: make(map[uuid.UUID][]*websocket.Conn),
		svc:                 svc,
	}
}

func (s *appManagerOrder) Register(
	ctx context.Context,
	connectionID uuid.UUID,
	conn *websocket.Conn,
	isShop bool,
) error {

	s.mu.Lock()

	if isShop {
		s.shopConnections[connectionID] = append(s.shopConnections[connectionID], conn)
	} else {
		s.customerConnections[connectionID] = append(s.customerConnections[connectionID], conn)
	}

	s.mu.Unlock()

	orders, err := s.svc.SearchActiveOrdersByCriteria(ctx, connectionID)
	if err != nil {
		if !errors.Is(err, errcustom.ErrNotFound) {
			slog.Error("Error buscando órdenes abiertas del shop",
				"shopID", connectionID,
				"error", err,
			)
			return err
		}

		slog.Info("No hay órdenes abiertas para este shop",
			"shopID", connectionID,
		)
	}

	for _, v := range orders {
		if err := s.BroadcastOrder(v); err != nil {
			slog.Error("Error enviando orden abierta al shop",
				"shopID", connectionID,
				"error", err,
			)
		}
	}

	go s.ping(conn)

	return nil
}

func (s *appManagerOrder) Unregister(connectionID uuid.UUID, isShop bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if isShop {
		delete(s.shopConnections, connectionID)
		slog.Info("Conexiones cerradas para el local",
			"shopID", connectionID)
	} else {
		delete(s.customerConnections, connectionID)
		slog.Info("Conexiones cerradas para el cliente",
			"customerID", connectionID)
	}
}

func (s *appManagerOrder) BroadcastOrder(order models.Order) error {
	data, err := json.Marshal(order)
	if err != nil {
		return err
	}

	s.mu.RLock()
	shopConns := s.shopConnections[order.ShopID]
	customerConns := s.customerConnections[order.CustomerID]
	s.mu.RUnlock()

	sendToConns := func(conns []*websocket.Conn, label string) {
		for _, conn := range conns {
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				slog.Error("Error enviando mensaje",
					"to", label,
					"id", order.ShopID,
					"error", err,
				)
				continue
			}
		}
		if len(conns) > 0 {
			slog.Info("Mensaje enviado",
				"to", label,
				"id", order.ShopID,
				"total", len(conns),
			)
		}
	}

	sendToConns(shopConns, "shop")

	sendToConns(customerConns, "customer")

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
