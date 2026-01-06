package main

import (
	"fmt"
	"learn-pub-sub-starter/internal/gamelogic"
	"learn-pub-sub-starter/internal/pubsub"
	"learn-pub-sub-starter/internal/routing"
  "time"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
  const connString = "amqp://guest:guest@localhost:5672/"
  fmt.Println("RabbitMQ connection string:", connString)
  conn, err := amqp.Dial(connString)
  if err != nil {
    // handle error
    fmt.Println("Error opening connection: ", err)
    return
  }
  defer conn.Close()
  fmt.Println("Peril game server connected to RabbitMQ!")
  gamelogic.PrintServerHelp()
  chann, err := conn.Channel()
  if err != nil {
    errf := fmt.Errorf("%v", err)

    fmt.Println("Error creating channel")
    fmt.Printf("Error: %s\n", errf)
    return
  }
  rGameLog := routing.GameLog{
    CurrentTime:   time.Now(),
    Message:       "Blank",
    Username:      "Uknown",
  }
  err = pubsub.PublishJSON(
    chann,
    string(routing.ExchangePerilTopic),
    string(routing.GameLogSlug),
    rGameLog,  
    )
  if err != nil {
    fmt.Println("Error Publishing GameLogSlug: ", err)
  }
  for {
    slice := gamelogic.GetInput()
    if len(slice) == 0 {
      continue
    }
    cmd := gamelogic.ParseCommand(slice[0])
    switch cmd {
    case gamelogic.CommandPause:
      err = pubsub.PublishJSON(
        chann, 
        routing.ExchangePerilDirect, 
        routing.PauseKey, 
        routing.PlayingState{
          IsPaused: true,
        },
        )
      if err != nil {
        errf := fmt.Errorf("%v", err)
        fmt.Println("Error publishing")
        fmt.Printf("Error: %s\n", errf)
        return
      }
    case gamelogic.CommandResume:
      err = pubsub.PublishJSON(
        chann, 
        routing.ExchangePerilDirect, 
        routing.PauseKey, 
        routing.PlayingState{
          IsPaused: false,
        },
        )
      if err != nil {
        errf := fmt.Errorf("%v", err)
        fmt.Println("Error publishing")
        fmt.Printf("Error: %s\n", errf)
        return
      }
    case gamelogic.CommandQuit:
      fmt.Println("Server Quiting!!!")
      return
    default:
      fmt.Println("Unknown command:", slice)
    }
  }
}
