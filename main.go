package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	MongoURI   = flag.String("mongouri", "mongodb://localhost:27017", "test_database")
	collection = flag.String("CRUD Database", "test", " for example")
	DBClient   *mongo.Client
	AppDB      *mongo.Database
)

//struct for broacasting data
type Data struct {
	Name string `json:"name"`
	Age  string `json:"age"`
}

type User struct {
	Name string `bson:"name"`
	Age  string `bson:"age"`
}

var clients = make(map[*websocket.Conn]bool) //all the client joined websocket
var broadcast = make(chan *Data)             //creating channel with user struct type
var broadcastUser = make(chan *User)         // creating channel with user struct type

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
	log.Infof("Number of Client %v", len(clients))
	for {
		// read in a message
		messageType, p, err := ws.ReadMessage()
		if err != nil {
			log.Infof("read message error %v", err)
			log.Info("Client left ")
			log.Infof("Number of client left %v", len(clients)-1)
			return
		}
		// print out that message for clarity
		fmt.Println(string(p), messageType)

	}

}

func main() {
	flag.Parse()

	DBClient = DBConnect(*MongoURI)
	if DBClient != nil {
		AppDB = DBClient.Database(*collection)
	}

	r := mux.NewRouter()

	r.HandleFunc("/", Home).Methods("GET")
	r.HandleFunc("/req", HandleUpcomingReq).Methods("POST")
	//r.HandleFunc("/getuser", HandleGetUser).Methods("GET")
	r.HandleFunc("/ws", HandleWs)

	go HandleGetUser()

	go HandleBrodcast()
	go HandleUserBrodcast()

	log.Fatal(http.ListenAndServe(":8080", r))
}

func DBConnect(URI string) *mongo.Client {
	clientOptions := options.Client().ApplyURI(URI)
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatalf("Problem connecting MongoURI %v, error : %v ", URI, err)
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatalf("Problem pinging server %v, err %v", URI, err)
	} else {
		log.Infof("Connected to Mongodb server %v", URI)
	}

	return client
}

func Home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Home Page")
}

//this function will broadcast data to every client whoever connected
func writer(info *Data) {
	broadcast <- info
}

func userWriter(user *User) {
	broadcastUser <- user
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

func HandleGetUser() {

	ticker := time.NewTicker(1 * time.Second)
	for _ = range ticker.C {
		var user User
		cur, err := AppDB.Collection("websocket").Find(context.TODO(), bson.M{})

		for cur.Next(context.TODO()) {
			err = cur.Decode(&user)
			if err != nil {
				log.Infof("Problem decoding user %v", err)
			} else {
				fmt.Println(user)
				userWriter(&user)
			}
		}

		cur.Close(context.TODO())
		//json.NewEncoder(w).Encode(user)
	}
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
				log.Printf("Websocket error handling data: %s", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func HandleUserBrodcast() {
	var user User

	for {
		val := <-broadcastUser
		user = User{Name: val.Name, Age: val.Age}
		for client := range clients {
			err := client.WriteJSON(user)
			if err != nil {
				log.Printf("Websocket error handling user: %s", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
