package main

import (
	"log"
	"net/http"
	"github.com/gorilla/websocket"
)

type data struct {
	X int
	Y int
	SessionId int
	Method string
}

type initSessionId struct{
	Num int
}

//global variables for unique users's identifiers is bad idea
//but it works for very simple tasks
//but better idea is to create queue for users	
var usersCounter = 0	

//store active connections
var clients = make(map[*websocket.Conn]bool)

//for easy message-data redirects
var broadcast = make(chan data)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	fs := http.FileServer(http.Dir("./"))
	http.Handle("/", fs)

	http.HandleFunc("/ws", handleConnections)

	// Start listening for mouse location updates
	go handleUpdates()

	log.Println("localhost:4567 listening")
	err := http.ListenAndServe("localhost:4567", nil)
	if err != nil {
		log.Fatal("ListenAndServe err: ", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	//new connection - new user
	usersCounter += 1

	//need to know exactly this users identifier
	//to be sure we won't delete cursor of the last user
	//when we delete random user
	currentUser := usersCounter

	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Make sure we close the connection when the function returns
	defer socket.Close()


	_, reply, err := socket.ReadMessage()

	if err == nil && string(reply) == "Hello" {

		//now we init user with his personal sessionId
		var initUser = initSessionId{usersCounter}

		if err = socket.WriteJSON(initUser); err != nil {

			log.Printf("Error while sending message to %d", usersCounter)
			usersCounter -= 1
			return 
		}

		//register new user
		log.Printf("new connection %d", usersCounter)
		clients[socket] = true	
	}

	for {
		var coordinates data
		err := socket.ReadJSON(&coordinates) 

		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, socket)
			//tell all other users to delete location of this user, when he close the page
			broadcast <- data{0, 0, currentUser, "leave"}
			break
		}
		broadcast <- data{coordinates.X, coordinates.Y, coordinates.SessionId, "move"}
	}
}

func handleUpdates() {
	for {
		newData := <-broadcast
		// Send update to all connected users
		for client := range clients {
			err := client.WriteJSON(newData)
			if err != nil {
				log.Printf("error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
