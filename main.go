package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

//struct for broacasting data
type Data struct {
	Name string `json:"name"`
	Age  string `json:"age"`
}

var clients = make(map[*websocket.Conn]bool) //all the client joined websocket
var broadcast = make(chan *Data)             // creating channel with Data struct type

//defining the websocket buffer,
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

//this function will start the websocket
func HandleWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}

	clients[ws] = true //mapping clients
	fmt.Println("Number of client connected ", len(clients))
	for {
		// read in a message
		messageType, p, err := ws.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		// print out that message for clarity
		fmt.Println(string(p), messageType)

	}

}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/", Home).Methods("GET")
	r.HandleFunc("/req", HandleUpcomingReq).Methods("POST")
	r.HandleFunc("/ws", HandleWs)
	go HandleBrodcast()

	log.Fatal(http.ListenAndServe(":8080", r))
}

func Home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Home Page")
}

//this function will broadcast data to every client whoever connected
func writer(info *Data) {
	broadcast <- info
}

//if we want to get data from mongo, we would have get the data in this function
func HandleUpcomingReq(w http.ResponseWriter, r *http.Request) {
	var info Data
	err := json.NewDecoder(r.Body).Decode(&info)
	if err != nil {
		fmt.Println("error while decoding the json")
		return
	}

	defer r.Body.Close()
	//we have the data, now we will passing data to channels
	go writer(&info)
}

func HandleBrodcast() {
	var data Data
	//infinite loop to get all the values from all the channels and broadcast data to
	//all connected client
	for {
		val := <-broadcast
		data = Data{Name: val.Name, Age: val.Age}
		fmt.Println(data)
		for client := range clients {
			err := client.WriteJSON(data)
			if err != nil {
				log.Printf("Websocket error b: %s", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
