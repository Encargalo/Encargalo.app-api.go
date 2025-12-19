package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type AppManagerOrder interface {
	Register(ctx context.Context, connectionID uuid.UUID, conn *websocket.Conn, isShop bool) error
	Unregister(connectionID uuid.UUID, conn *websocket.Conn, isShop bool)
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

func (s *appManagerOrder) Register(ctx context.Context, connectionID uuid.UUID, conn *websocket.Conn, isShop bool) error {

	s.mu.Lock()
	if isShop {
		s.shopConnections[connectionID] = append(s.shopConnections[connectionID], conn)
	} else {
		s.customerConnections[connectionID] = append(s.customerConnections[connectionID], conn)
	}
	s.mu.Unlock()

	// ... (Tu lógica de BroadcastOrder y búsqueda de órdenes se mantiene igual) ...

	// SOLUCIÓN 1: Iniciamos el Ping (solo escritura)
	go s.writePump(conn)

	return nil
}

// SOLUCIÓN 2: Arreglamos Unregister para borrar SOLO la conexión específica
func (s *appManagerOrder) Unregister(connectionID uuid.UUID, conn *websocket.Conn, isShop bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var connections map[uuid.UUID][]*websocket.Conn
	if isShop {
		connections = s.shopConnections
	} else {
		connections = s.customerConnections
	}

	if conns, ok := connections[connectionID]; ok {
		// Filtramos para eliminar solo esta conexión específica
		filtered := conns[:0]
		for _, c := range conns {
			if c != conn {
				filtered = append(filtered, c)
			}
		}

		if len(filtered) == 0 {
			delete(connections, connectionID)
		} else {
			connections[connectionID] = filtered
		}
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

// Renombramos ping a writePump para ser más claros. Solo escribe.
func (s *appManagerOrder) writePump(conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			return // Si falla escribir, salimos. El ReadLoop del handler detectará el cierre.
		}
	}
}
