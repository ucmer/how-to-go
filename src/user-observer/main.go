package main

import (
	"encoding/json"
	"excample/user-observer/async-base"
	"github.com/gin-gonic/gin"
	"github.com/streadway/amqp"
	"log"
	"net/http"
)

// user represents data about a user.
type user struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Points   int    `json:"points"`
	Division string `json:"division"`
	Team     string `json:"team"`
}

type observer struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	queue      amqp.Queue
}

var users = []user{
	{ID: "1", Name: "First User", Role: "Admin", Division: "development", Team: "", Points: 5},
	{ID: "2", Name: "Second User", Role: "Employee", Division: "development", Team: "Rockets", Points: 100},
	{ID: "3", Name: "Third User", Role: "Manager", Division: "development", Team: "Scyfall", Points: 10},
}

func main() {
	router := gin.Default()
	router.GET("/users", getUsers)
	router.GET("/user/:id", getUserByID)

	observer := initObserver()

	awaiter := async_base.Exec(func() interface{} {
		return observeUserQueue(observer)
	})
	router.Run("localhost:8090")

	awaiter.Await()
}

func getUserByID(context *gin.Context) {
	id := context.Param("id")

	// Loop over the list of users, looking for
	// an user whose ID value matches the parameter.
	for _, user := range users {
		if user.ID == id {
			context.IndentedJSON(http.StatusOK, user)
			return
		}
	}
	context.IndentedJSON(http.StatusNotFound, gin.H{"message": "user not found"})
}

func getUsers(context *gin.Context) {
	context.IndentedJSON(http.StatusOK, users)
}

func observeUserQueue(userObserver observer) bool {
	var newUser user

	msgs, err := userObserver.channel.Consume(
		userObserver.queue.Name, // queue
		"",                      // consumer
		true,                    // auto-ack
		false,                   // exclusive
		false,                   // no-local
		false,                   // no-wait
		nil,                     // args
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			json.Unmarshal(d.Body, &newUser)
			users = append(users, newUser)
			log.Printf("Received a message: %s", d.Body)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever

	return true
}

func initObserver() observer {
	var userPublisher observer

	// connect to Rabbit
	conn, err := amqp.Dial("amqp://rhenus-digital:rhenus-digital@message-hub.rhenus-digital.local:30346")
	failOnError(err, "Failed to connect to RabbitMQ")

	// create channel
	channel, err := conn.Channel()
	failOnError(err, "Failed to open a channel")

	// declare queue
	queue, err := channel.QueueDeclare(
		"booking.appointments", // name
		true,                   // durable
		false,                  // delete when unused
		false,                  // exclusive
		false,                  // no-wait
		nil,                    // arguments
	)
	failOnError(err, "Failed to declare a queue")

	userPublisher.connection = conn
	userPublisher.channel = channel
	userPublisher.queue = queue

	return userPublisher
}

func destructPublisher(userPublisher observer) {
	defer userPublisher.connection.Close()
	defer userPublisher.channel.Close()
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}
