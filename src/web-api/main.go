package main

import (
	"encoding/json"
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

type publisher struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	queue      amqp.Queue
}

func postUser(context *gin.Context) {
	var newUser user
	var publisher = initPublisher()

	// Call BindJSON to bind the received JSON to
	// newUser.
	if err := context.BindJSON(&newUser); err != nil {
		return
	}

	// Add the new user to the slice.
	publishUserToQueue(publisher, newUser)
	//users = append(users, newUser)
	context.IndentedJSON(http.StatusCreated, newUser)

	destructPublisher(publisher)
}

func main() {
	router := gin.Default()
	router.POST("/user", postUser)

	router.Run("localhost:8080")
}

func initPublisher() publisher {
	var userPublisher publisher

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

func destructPublisher(userPublisher publisher) {
	defer userPublisher.connection.Close()
	defer userPublisher.channel.Close()
}

func publishUserToQueue(userPublisher publisher, newUser user) {
	var body, err = json.Marshal(newUser)
	failOnError(err, "Failed to serialize user")

	err = userPublisher.channel.Publish(
		"booking",                // exchange
		userPublisher.queue.Name, // routing key
		false,                    // mandatory
		false,                    // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        body,
		})
	failOnError(err, "Failed to publish a message")
	log.Printf(" [x] Sent %s\n", body)
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}
