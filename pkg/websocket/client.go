package websocket

import "github.com/gorilla/websocket"

type Client struct {
	Conn     *websocket.Conn
	Inbound  chan []byte
	Outbound chan []byte
}

func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		Conn:     conn,
		Inbound:  make(chan []byte),
		Outbound: make(chan []byte),
	}
}

func (c *Client) Read() {
	defer c.Conn.Close()
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
		println(string(message))
		c.Inbound <- message
	}
}

func (c *Client) Write() {
	defer func() {
		c.Conn.Close()
	}()
	for message := range c.Outbound {
		if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			break
		}
	}
}
