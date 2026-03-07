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
	Register(ctx context.Context, connectionID uuid.UUID, conn *websocket.Conn, isShop bool) (*WSClient, error)
	Unregister(connectionID uuid.UUID, client *WSClient, isShop bool)
	BroadcastOrder(order models.Order) error
}

type WSClient struct {
	conn  *websocket.Conn
	send  chan []byte
	ready chan struct{}
}

type appManagerOrder struct {
	mu                  sync.RWMutex
	shopConnections     map[uuid.UUID][]*WSClient
	customerConnections map[uuid.UUID][]*WSClient
	svc                 ports.OrdersApp
}

func NewManagerOrderApp(svc ports.OrdersApp) AppManagerOrder {
	return &appManagerOrder{
		shopConnections:     make(map[uuid.UUID][]*WSClient),
		customerConnections: make(map[uuid.UUID][]*WSClient),
		svc:                 svc,
	}
}

func (s *appManagerOrder) Register(
	ctx context.Context,
	connectionID uuid.UUID,
	conn *websocket.Conn,
	isShop bool,
) (*WSClient, error) {

	client := &WSClient{
		conn:  conn,
		send:  make(chan []byte, 256),
		ready: make(chan struct{}),
	}

	s.mu.Lock()
	if isShop {
		s.shopConnections[connectionID] = append(s.shopConnections[connectionID], client)
	} else {
		s.customerConnections[connectionID] = append(s.customerConnections[connectionID], client)
	}
	s.mu.Unlock()

	go s.writePump(client)

	orders, err := s.svc.SearchActiveOrdersByCriteria(ctx, connectionID)
	if err != nil && !errors.Is(err, errcustom.ErrNotFound) {
		return nil, err
	}

	for _, order := range orders {
		data, _ := json.Marshal(order)
		client.send <- data
	}

	close(client.ready)

	return client, nil
}

func (s *appManagerOrder) Unregister(
	connectionID uuid.UUID,
	client *WSClient,
	isShop bool,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var connections map[uuid.UUID][]*WSClient
	if isShop {
		connections = s.shopConnections
	} else {
		connections = s.customerConnections
	}

	if list, ok := connections[connectionID]; ok {
		filtered := list[:0]
		for _, c := range list {
			if c != client {
				filtered = append(filtered, c)
			}
		}

		if len(filtered) == 0 {
			delete(connections, connectionID)
		} else {
			connections[connectionID] = filtered
		}
	}

	close(client.send)
}

func (s *appManagerOrder) BroadcastOrder(order models.Order) error {
	data, err := json.Marshal(order)
	if err != nil {
		return err
	}

	s.mu.RLock()
	shopClients := s.shopConnections[order.ShopID]
	customerClients := s.customerConnections[order.CustomerID]
	s.mu.RUnlock()

	send := func(clients []*WSClient) {
		for _, c := range clients {
			select {
			case <-c.ready:
				select {
				case c.send <- data:
				default:
					slog.Warn("Cliente lento, mensaje descartado")
				}
			default:
			}
		}
	}

	send(shopClients)
	send(customerClients)

	return nil
}

func (s *appManagerOrder) writePump(c *WSClient) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *WSClient) Send(msg []byte) {
	select {
	case <-c.ready:
		select {
		case c.send <- msg:
		default:
		}
	default:
	}
}
