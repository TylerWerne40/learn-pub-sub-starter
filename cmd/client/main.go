package main

import (
	"context"
	"fmt"
	"learn-pub-sub-starter/internal/gamelogic"
	"learn-pub-sub-starter/internal/pubsub"
	"learn-pub-sub-starter/internal/routing"
	"strconv"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
  ctx := context.Background()
  const connString = "amqp://guest:guest@localhost:5672/"
  fmt.Println("RabbitMQ connection string:", connString)
  conn, err := amqp.Dial(connString)
  if err != nil {
    // handle error
    fmt.Println("Error opening connection: ", err)
    return
  }
  defer conn.Close()
  username, err := gamelogic.ClientWelcome()
  pause_chan, _, err := pubsub.DeclareAndBind(conn, 
    routing.ExchangePerilDirect, 
    routing.PauseKey + "." + username, 
    routing.PauseKey, 
    "transient")
  gs := gamelogic.NewGameState(username)
  move_chan, move_queue, err := pubsub.DeclareAndBind(conn,
    routing.ExchangePerilTopic,
    routing.ArmyMovesPrefix + "." + username,
    routing.ArmyMovesPrefix,
    pubsub.Transient)
  pubsub.SubscribeJSON(
    conn,
    routing.ExchangePerilTopic,
    move_queue.Name,
    routing.ArmyMovesPrefix + ".*",
    pubsub.Transient,
    HandlerMove(gs, move_chan),
    )
  war_chan, _, err := pubsub.DeclareAndBind(
    conn,
    routing.ExchangePerilTopic,
    "war",
    routing.WarRecognitionsPrefix + ".#", 
    pubsub.Durable,
    )
  if err != nil {
    fmt.Println("war subscribe error:", err)
  }  
  pubsub.SubscribeJSON(
    conn,
    routing.ExchangePerilTopic,
    "war",
    routing.WarRecognitionsPrefix + ".#",
    pubsub.Durable,
    HandlerWar(ctx, gs, war_chan),
    )
  for {
    pubsub.SubscribeJSON(
      conn, 
      routing.ExchangePerilDirect,
      routing.PauseKey + "." + username,
      routing.PauseKey,
      "transient",
      HandlerPause(gs, pause_chan),
    )
    slice := gamelogic.GetInput()
    if len(slice) == 0 {
      continue
    }
    cmd := gamelogic.ParseCommand(slice[0])
    switch cmd {
      case gamelogic.CommandSpawn:
        if len(slice) < 3 {
          fmt.Println("Invalid command call")
        }
        err := gs.CommandSpawn(slice)
        if err != nil {
          fmt.Println("Error Spawning unit: ", err)
        }
      case gamelogic.CommandMove:
        _, err := gs.CommandMove(slice) // ignore outcome for now
        if err != nil {
          fmt.Println("Move Command incorrect:", err)
        }
        // assume valid move
        units := make([]gamelogic.Unit, 0)
        for _, id := range slice[2:] {  
          unit_id, _ := strconv.Atoi(id)
          unit, _ := gs.GetUnit(unit_id)
          units = append(units, unit)
        }
        if len(units) == 0 {
          fmt.Println("No valid units to move")
          continue
        }
        moveToPublish := gamelogic.ArmyMove{
          Player: gs.Player,
          Units: units,
          ToLocation: gamelogic.Location(slice[1]),
        }
        err = pubsub.PublishJSON(
          move_chan,
          string(routing.ExchangePerilTopic),
          string(routing.ArmyMovesPrefix) + "." + username,
          moveToPublish,
        )
        if err != nil {
          fmt.Println("Move could not be published")
          continue
        }
        fmt.Println("Move was Published!")
      case gamelogic.CommandHelp:
        gamelogic.PrintClientHelp()
      case gamelogic.CommandSpam:
        fmt.Println("Spamming not allowed yet!")
      case gamelogic.CommandQuit:
        gamelogic.PrintQuit()
        return
      default:
        fmt.Println("Unknown Command: ", slice)
        continue

  }
  }
}
