package ws

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WsManager 管理所有 WebSocket 连接
type WsManager struct {
	Clients    map[*websocket.Conn]bool
	Register   chan *websocket.Conn
	Unregister chan *websocket.Conn
	Broadcast  chan []byte
	Lock       sync.Mutex
}

var Manager = WsManager{
	Clients:    make(map[*websocket.Conn]bool),
	Register:   make(chan *websocket.Conn),
	Unregister: make(chan *websocket.Conn),
	Broadcast:  make(chan []byte),
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (manager *WsManager) Start() {
	for {
		select {
		case conn := <-manager.Register:
			manager.Lock.Lock()
			manager.Clients[conn] = true
			manager.Lock.Unlock()
			fmt.Println("新用户连接，当前在线:", len(manager.Clients))

		case conn := <-manager.Unregister:
			manager.Lock.Lock()
			if _, ok := manager.Clients[conn]; ok {
				delete(manager.Clients, conn)
				conn.Close()
			}
			manager.Lock.Unlock()
			fmt.Println("用户断开，当前在线:", len(manager.Clients))

		case message := <-manager.Broadcast:
			manager.Lock.Lock()
			for conn := range manager.Clients {
				err := conn.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					conn.Close()
					delete(manager.Clients, conn)
				}
			}
			manager.Lock.Unlock()
		}
	}
}

func WsHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	Manager.Register <- conn

	go func() {
		defer func() {
			Manager.Unregister <- conn
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

func SendHeartbeat() {
	for {
		time.Sleep(30 * time.Second)
		Manager.Broadcast <- []byte("ping")
	}
}
