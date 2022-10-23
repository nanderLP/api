// Modified version of https://github.com/fasthttp/websocket/blob/master/_examples/chat/client.go
// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license
// You can find it here: https://github.com/fasthttp/websocket/blob/master/LICENSE

package websocket

import (
	"bytes"
	"encoding/json"
	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	fiberWebsocket "github.com/gofiber/websocket/v2"
	"log"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// used to differenciate between clients
	id string

	// Buffered channel of outbound messages.
	send chan Message
}

// ClientMessage representates a message received from the Client
type ClientMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type Message struct {
	Id      string        `json:"id"`
	Message ClientMessage `json:"message"`
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		mt, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		clientMessage := ClientMessage{}
		err = json.Unmarshal(message, &clientMessage)
		c.hub.broadcast <- Message{
			Id:      c.id,
			Message: clientMessage,
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			jsonMessage, err := json.Marshal(message)
			if err != nil {
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(jsonMessage)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				jsonMessage, err := json.Marshal(<-c.send)
				if err != nil {
					return
				}
				w.Write(jsonMessage)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub) fiber.Handler {
	return fiberWebsocket.New(func(c *fiberWebsocket.Conn) {
		id := utils.CopyString(c.Query("id"))
		if id == "" {
			c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
		client := &Client{hub: hub, id: id, conn: c.Conn, send: make(chan Message, 256)}
		client.hub.register <- client

		go client.writePump()
		// I spent 2 days debugging this shit and it seems like I can't start readPump as a goroutine
		client.readPump()
	})
}
