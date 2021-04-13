package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"net/http"
	"sync"
	"time"
)

type Connection struct {
	Socket *websocket.Conn
	mu     sync.Mutex
}

type Message struct {
	ChatID    string  `json:"chat_id"`
	Message   string  `json:"message"`
	Timestamp int64 `json:"timestamp"`
}

var _ = godotenv.Load()

var broadcast = make(chan Message)
var channels = make(map[string]map[*Connection]bool)
var initializer = websocket.Upgrader{ CheckOrigin: func(r *http.Request) bool { return true } }

func main() {
	instance := echo.New()

	instance.Use(middleware.Logger())
	instance.Use(middleware.Recover())

	instance.Add("GET","/message-slot/:id", slotHandler)

	go handleMessages()

	instance.Logger.Fatal(instance.Start(":1002"))
}

func slotHandler(cnt echo.Context) error {
	chatId := cnt.Param("id")
	socket, err := initializer.Upgrade(cnt.Response(), cnt.Request(), nil)

	connection := new(Connection)
	connection.Socket = socket

	var msg Message

	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	defer func() {
		err := connection.Socket.Close()

		if err != nil {
			cnt.Logger().Error(err)
		}
	}()

	if channels[chatId] == nil {
		channels[chatId] = make(map[*Connection]bool)
	}
	channels[chatId][connection] = true

	for {
		err = connection.Socket.ReadJSON(&msg)

		if err != nil {
			cnt.Logger().Error(err)
			delete(channels[chatId], connection)
			break
		}
		msg.ChatID = chatId
		msg.Timestamp = time.Now().Unix()
		broadcast <- msg
	}

	return err
}

func handleMessages() {
	for {
		msg := <-broadcast
		chatId := msg.ChatID

		for client := range channels[chatId] {
			err := client.Send(msg)
			if err != nil {
				fmt.Printf("client write error: %v", err)
				err = client.Socket.Close()
				delete(channels[chatId], client)

				if err != nil {
					fmt.Printf("client close error: %v", err)
				}
			}
		}
	}
}

func (c *Connection) Send(message Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Socket.WriteJSON(message)
}