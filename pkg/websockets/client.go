package socket

import (
	"context"
	"easyflow-backend/pkg/database"
	"easyflow-backend/pkg/jwt"
	"easyflow-backend/pkg/logger"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

// Custom error types for better categorization
var (
	ErrConnectionClosed = errors.New("websocket connection closed")
	ErrRoomAccess       = errors.New("room access denied")
	ErrMessageTooLarge  = errors.New("message exceeds maximum size")
	ErrWriteTimeout     = errors.New("write operation timed out")
	ErrReadTimeout      = errors.New("read operation timed out")
	ErrInvalidMessage   = errors.New("invalid message format")
	ErrClientDisconnect = errors.New("client disconnected")
	ErrDBAccess         = errors.New("database access error")
)

// Connection states
const (
	stateConnected     = "connected"
	stateDisconnecting = "disconnecting"
	stateDisconnected  = "disconnected"
	stateReconnecting  = "reconnecting"
	stateError         = "error"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024 * 1024 // 1 MB
)

type clientMessage struct {
	Room string `json:"room"`
	Data string `json:"data"`
	Iv   string `json:"iv"`
}

type message struct {
	clientMessage
	SenderID string `json:"sender_id"`
}

type errorMessage struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

type clientStats struct {
	messagesReceived  int64
	messagesSent      int64
	errors            int64
	lastActivity      time.Time
	connectionStarted time.Time
	mutex             sync.Mutex
}

type Client struct {
	conn          *websocket.Conn
	connMutex     sync.RWMutex // Added mutex for connection access
	send          chan message
	payload       *jwt.JWTTokenPayload
	logger        *logger.Logger
	rooms         map[string]*Room
	roomsMutex    sync.RWMutex
	db            *gorm.DB
	ctx           context.Context
	cancel        context.CancelFunc
	state         string
	stateMutex    sync.RWMutex
	disconnectErr error // Stores the error that caused disconnection
	stats         clientStats
	cleanupOnce   sync.Once // Ensure cleanup runs exactly once
}

func newClient(conn *websocket.Conn, payload *jwt.JWTTokenPayload, hub *hub) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	c := &Client{
		conn:       conn,
		send:       make(chan message, 256), // Buffered channel to prevent blocking
		payload:    payload,
		logger:     hub.logger,
		rooms:      make(map[string]*Room),
		roomsMutex: sync.RWMutex{},
		db:         hub.db,
		ctx:        ctx,
		cancel:     cancel,
		state:      stateConnected,
		stats: clientStats{
			connectionStarted: time.Now(),
			lastActivity:      time.Now(),
		},
	}

	// Initialize rooms safely with proper error handling
	if err := c.initializeRooms(hub); err != nil {
		c.logError("Failed to initialize rooms", err)
		cancel() // Cancel context on initialization failure
		return nil
	}

	c.logger.PrintfInfo("Client %s initialized successfully", c.payload.UserID)
	return c
}

func (c *Client) initializeRooms(hub *hub) error {
	var chats []database.ChatsUsers

	err := hub.db.Find(&chats).Where("user_id = ?", c.payload.UserID).Error
	if err != nil {
		// Wrap database errors with context
		return fmt.Errorf("%w: %v", ErrDBAccess, err)
	}

	hub.roomsMutex.RLock()
	defer hub.roomsMutex.RUnlock()

	for _, chat := range chats {
		if room, exists := hub.rooms[chat.ChatID]; exists {
			room.addClient(c)
			c.logger.PrintfDebug("Client %s added to existing room %s", c.payload.UserID, chat.ChatID)
			continue
		}
		room := newRoom(chat.ChatID, hub)
		room.addClient(c)
		c.logger.PrintfDebug("Client %s added to new room %s", c.payload.UserID, chat.ChatID)
	}

	return nil
}

// Safe connection access methods
func (c *Client) getConn() *websocket.Conn {
	c.connMutex.RLock()
	defer c.connMutex.RUnlock()
	return c.conn
}

// initiateGracefulClose sends a proper close frame and waits for connection to close
func (c *Client) initiateGracefulClose(ctx context.Context, closeCode int, message string) {
	// Prevent duplicate close
	if !c.setState(stateDisconnecting) {
		return
	}

	c.logger.PrintfInfo("Initiating graceful close for user %s with code %d: %s",
		c.payload.UserID, closeCode, message)

	// Get connection safely
	conn := c.getConn()
	if conn == nil {
		c.logger.PrintfDebug("Connection already nil during graceful close for user %s", c.payload.UserID)
		c.cleanup()
		return
	}

	// Prepare close message
	closeMessage := websocket.FormatCloseMessage(closeCode, message)

	// Set write deadline
	err := conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err != nil {
		c.logError("Failed to set write deadline for close frame", err)
		c.cleanup() // Force immediate close
		return
	}

	// Send close frame
	err = conn.WriteMessage(websocket.CloseMessage, closeMessage)
	if err != nil {
		c.logError("Failed to send close frame", err)
		c.cleanup() // Force immediate close
		return
	}

	// Start a goroutine to wait for client to acknowledge close
	// or force close after timeout
	closeAckCh := make(chan struct{})

	// Set a read handler for close acknowledgment
	origHandler := conn.CloseHandler()
	conn.SetCloseHandler(func(code int, text string) error {
		// Run original handler
		if origHandler != nil {
			_ = origHandler(code, text)
		}

		c.logger.PrintfInfo("Received close acknowledgment from client %s: %d - %s",
			c.payload.UserID, code, text)

		close(closeAckCh)
		return nil
	})

	// Set read deadline slightly longer than the context timeout
	deadline, ok := ctx.Deadline()
	if ok {
		err := conn.SetReadDeadline(deadline.Add(500 * time.Millisecond))
		if err != nil {
			c.logger.PrintfError("Failed to set read deadline, %s", err)
		}
	} else {
		err := conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		if err != nil {
			c.logger.PrintfError("Failed to set read deadline, %s", err)
		}
	}

	// Wait for acknowledgment or timeout
	select {
	case <-closeAckCh:
		c.logger.PrintfInfo("Client %s acknowledged close", c.payload.UserID)
	case <-ctx.Done():
		c.logger.PrintfWarning("Timeout waiting for client %s to acknowledge close", c.payload.UserID)
	case <-time.After(3 * time.Second): // Safety timeout
		c.logger.PrintfWarning("Safety timeout reached while waiting for close acknowledgment")
	}

	// Trigger final cleanup
	c.cleanup()
}

// Helper method to safely change state
func (c *Client) setState(newState string) bool {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()

	// Only allow transitioning to disconnecting once
	if c.state == stateDisconnecting || c.state == stateDisconnected || c.state == stateError {
		return false
	}

	c.state = newState
	return true
}

func (c *Client) getState() string {
	c.stateMutex.RLock()
	defer c.stateMutex.RUnlock()
	return c.state
}

func (c *Client) logError(message string, err error) {
	// Track error statistics
	c.stats.mutex.Lock()
	c.stats.errors++
	c.stats.mutex.Unlock()

	c.logger.PrintfError("%s: %v [user:%s] [state:%s] [conn_uptime:%s]",
		message,
		err,
		c.payload.UserID,
		c.getState(),
		time.Since(c.stats.connectionStarted).String())
}

func (c *Client) readMessages() {
	var readErr error

	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("panic in readMessages: %v", r)
			}

			stack := make([]byte, 4096)
			stack = stack[:runtime.Stack(stack, false)]
			c.logError("Panic recovered in readMessages", fmt.Errorf("%v\n%s", err, stack))

			readErr = err
		}

		// Always log connection closure, regardless of how it happened
		if readErr != nil {
			if websocket.IsCloseError(readErr, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				c.logger.PrintfInfo("WebSocket connection closed normally for user %s", c.payload.UserID)
			} else {
				c.logger.PrintfWarning("WebSocket read loop terminated with error, %s", readErr)
			}
		} else {
			c.logger.PrintfInfo("WebSocket read loop terminated for user %s", c.payload.UserID)
		}

		// Set disconnect error if it's not already set
		if c.disconnectErr == nil {
			c.disconnectErr = readErr
		}

		// Call cleanup (protected by sync.Once)
		c.cleanup()
	}()

	// Get connection safely
	conn := c.getConn()
	if conn == nil {
		readErr = errors.New("connection is nil at read start")
		return
	}

	// Configure the WebSocket connection
	conn.SetReadLimit(maxMessageSize)
	conn.SetPongHandler(func(string) error {
		// Update last activity time on pong
		c.stats.mutex.Lock()
		c.stats.lastActivity = time.Now()
		c.stats.mutex.Unlock()

		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	// Initial read deadline
	if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		readErr = fmt.Errorf("failed to set initial read deadline: %w", err)
		return
	}

	// Main read loop
	for {
		select {
		case <-c.ctx.Done():
			// Context canceled, exit cleanly
			readErr = fmt.Errorf("read loop terminated due to context cancellation: %w", c.ctx.Err())
			return

		default:
			// Check if connection is still valid
			conn = c.getConn()
			if conn == nil {
				readErr = errors.New("connection became nil during read loop")
				return
			}

			// Read message with timeout
			var msg clientMessage
			if err := conn.ReadJSON(&msg); err != nil {
				// Check for different error types
				if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					readErr = fmt.Errorf("unexpected close error: %w", err)
				} else if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					readErr = ErrConnectionClosed
				} else {
					// Parse JSON errors
					var syntaxErr *json.SyntaxError
					var unmarshalTypeErr *json.UnmarshalTypeError

					switch {
					case errors.As(err, &syntaxErr):
						readErr = fmt.Errorf("%w: invalid JSON at position %d", ErrInvalidMessage, syntaxErr.Offset)
					case errors.As(err, &unmarshalTypeErr):
						readErr = fmt.Errorf("%w: wrong field type at position %d", ErrInvalidMessage, unmarshalTypeErr.Offset)
					default:
						readErr = fmt.Errorf("websocket read error: %w", err)
					}
				}
				return
			}

			// Message received successfully
			c.stats.mutex.Lock()
			c.stats.messagesReceived++
			c.stats.lastActivity = time.Now()
			c.stats.mutex.Unlock()

			c.logger.PrintfDebug("Received message from user %s for room %s", c.payload.UserID, msg.Room)

			// Process the message
			if err := c.handleMessage(msg); err != nil {
				readErr = fmt.Errorf("message handling error: %w", err)
				return
			}
		}
	}
}

func (c *Client) handleMessage(msg clientMessage) error {
	c.roomsMutex.RLock()
	defer c.roomsMutex.RUnlock()

	room, exists := c.rooms[msg.Room]
	if !exists {
		c.logger.PrintfWarning("Access to room: %s denied for user: %s", msg.Room, c.payload.UserID)

		// Send error response to client
		err := c.sendErrorMessage("Access Denied",
			fmt.Sprintf("You do not have permission to access room %s or it does not exist", msg.Room))
		if err != nil {
			return fmt.Errorf("%w: %v", ErrRoomAccess, err)
		}

		return nil // Return nil to keep connection alive
	}

	// Create full message with sender ID
	message := message{
		clientMessage: msg,
		SenderID:      c.payload.UserID,
	}

	// Marshal to JSON for Valkey publication
	jsonBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error serializing message: %w", err)
	}

	// Publish to Valkey
	pubCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := room.hub.valkey.B().Publish().
		Channel(fmt.Sprintf("room-%s", room.id)).
		Message(string(jsonBytes)).
		Build()

	if err := room.hub.valkey.Do(pubCtx, cmd).Error(); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func (c *Client) sendErrorMessage(errType string, details string) error {
	errorMsg := errorMessage{
		Error:   errType,
		Details: details,
	}

	conn := c.getConn()
	if conn == nil {
		return errors.New("connection is nil, cannot send error message")
	}

	err := conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}

	if err := conn.WriteJSON(errorMsg); err != nil {
		return fmt.Errorf("failed to write error message: %w", err)
	}

	return nil
}

func (c *Client) writeMessages() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		if r := recover(); r != nil {
			stack := make([]byte, 4096)
			stack = stack[:runtime.Stack(stack, false)]
			c.logger.PrintfWarning("Panic recovered in writeMessages, %s", fmt.Errorf("%v\n%s", r, stack))
		}

		// Log write loop termination
		c.logger.PrintfInfo("Write loop terminated for user %s", c.payload.UserID)

		// Call cleanup
		c.cleanup()
	}()

	for {
		select {
		case <-c.ctx.Done():
			// Context was canceled, exit gracefully
			c.logger.PrintfDebug("Write loop terminated by context for user %s", c.payload.UserID)
			return

		case msg, ok := <-c.send:
			if !ok {
				// Channel was closed, terminate the goroutine
				c.logger.PrintfDebug("Send channel closed for user %s", c.payload.UserID)
				return
			}

			// Get connection safely
			conn := c.getConn()
			if conn == nil {
				c.logger.PrintfDebug("Connection is nil for user %s, stopping write loop", c.payload.UserID)
				return
			}

			// Set write deadline
			if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				c.logError("Failed to set write deadline", err)
				return
			}

			if err := conn.WriteJSON(msg); err != nil {
				c.logError("Failed to write message", err)
				return
			}

			// Update stats
			c.stats.mutex.Lock()
			c.stats.messagesSent++
			c.stats.lastActivity = time.Now()
			c.stats.mutex.Unlock()

		case <-ticker.C:
			// Get connection safely
			conn := c.getConn()
			if conn == nil {
				c.logger.PrintfDebug("Connection is nil for user %s, stopping write loop", c.payload.UserID)
				return
			}

			if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				c.logError("Failed to set write deadline for ping", err)
				return
			}

			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					c.logger.PrintfInfo("Connection closed during ping for user %s", c.payload.UserID)
				} else {
					c.logError("Failed to write ping", err)
				}
				return
			}
		}
	}
}

func (c *Client) cleanup() {
	// Use the cleanupOnce to ensure cleanup happens only once
	c.cleanupOnce.Do(func() {
		c.logger.PrintfInfo("Starting cleanup for user %s", c.payload.UserID)

		// First cancel context to signal all goroutines
		c.cancel()

		// Mark client as disconnecting
		c.setState(stateDisconnecting)

		// Get and clear the connection atomically
		c.connMutex.Lock()
		connToClose := c.conn
		c.conn = nil
		c.connMutex.Unlock()

		// Close the connection if it exists
		if connToClose != nil {
			// Try to send a close frame if not already sent
			// This is a best-effort attempt, ignoring errors
			closeMessage := websocket.FormatCloseMessage(
				websocket.CloseGoingAway,
				"Server closing connection")

			_ = connToClose.SetWriteDeadline(time.Now().Add(100 * time.Millisecond))
			_ = connToClose.WriteMessage(websocket.CloseMessage, closeMessage)

			// Close the underlying connection
			if err := connToClose.Close(); err != nil {
				if !errors.Is(err, websocket.ErrCloseSent) {
					c.logger.PrintfError("Error closing connection for user %s: %v",
						c.payload.UserID, err)
				}
			}
		}

		// Lock room mutex before accessing rooms
		c.roomsMutex.Lock()

		// Remove client from all rooms
		for roomID, room := range c.rooms {
			c.logger.PrintfDebug("Removing user %s from room %s", c.payload.UserID, roomID)
			room.removeClient(c)
		}

		// Clear the rooms map
		c.rooms = nil
		c.roomsMutex.Unlock()

		// Close send channel safely if not already closed
		select {
		case _, ok := <-c.send:
			if ok {
				close(c.send)
			}
		default:
			close(c.send)
		}
		c.logger.PrintfDebug("Closed send channel for user %s", c.payload.UserID)

		// IMPORTANT: Always log connection stats
		c.logger.PrintfInfo("User %s disconnected. Stats: received=%d sent=%d errors=%d uptime=%s",
			c.payload.UserID,
			c.stats.messagesReceived,
			c.stats.messagesSent,
			c.stats.errors,
			time.Since(c.stats.connectionStarted).String())

		c.setState(stateDisconnected)
	})
}
