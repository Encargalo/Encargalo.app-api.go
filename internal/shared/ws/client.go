package ws

import (
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	readTimeout  = 60 * time.Second
	writeTimeout = 10 * time.Second
	pingInterval = 30 * time.Second
	readLimit    = 4096
)

// Client is a buffered WebSocket writer that satisfies both ports.RiderConn and ports.ManagerConn.
type Client struct {
	conn      *websocket.Conn
	send      chan []byte
	closeOnce sync.Once
}

func NewClient(conn *websocket.Conn) *Client {
	conn.SetReadLimit(readLimit)
	_ = conn.SetReadDeadline(time.Now().Add(readTimeout))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(readTimeout))
		return nil
	})
	return &Client{
		conn: conn,
		send: make(chan []byte, 256),
	}
}

func (c *Client) Send(msg []byte) {
	select {
	case c.send <- msg:
	default:
		slog.Warn("ws client slow, message dropped")
	}
}

func (c *Client) Close() {
	c.closeOnce.Do(func() { close(c.send) })
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
