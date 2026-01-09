package main

import (
	"fmt"
	"learn-pub-sub-starter/internal/gamelogic"
	"learn-pub-sub-starter/internal/pubsub"
	"learn-pub-sub-starter/internal/routing"
  amqp "github.com/rabbitmq/amqp091-go"
)


func HandlerPause(gs *gamelogic.GameState, publishCh *amqp.Channel) func(routing.PlayingState) pubsub.AckType{
  return func(ps routing.PlayingState) pubsub.AckType {
    fmt.Print("> ")
    gs.HandlePause(ps)
    return pubsub.Ack
  }
}
func HandlerMove(gs *gamelogic.GameState, publishCh *amqp.Channel) func(gamelogic.ArmyMove) pubsub.AckType {
  return func(move gamelogic.ArmyMove) pubsub.AckType {
    moveOutcome := gs.HandleMove(move)
    fmt.Print("\n> ")
    switch moveOutcome {
      case gamelogic.MoveOutcomeMakeWar:
        recognition := gamelogic.RecognitionOfWar{
          Attacker: move.Player,
          Defender: gs.GetPlayerSnap(),
          }
        err := pubsub.PublishJSON(
          publishCh,
          routing.ExchangePerilTopic,
          routing.WarRecognitionsPrefix+"."+gs.GetUsername(),
          recognition,
          )
        if err != nil {
          fmt.Printf("Publish error: %v\n", err)
          return pubsub.NackRequeue
        }

        return pubsub.NackRequeue

      case gamelogic.MoveOutComeSafe:
        return pubsub.Ack
      case gamelogic.MoveOutcomeSamePlayer:
        return pubsub.NAckDiscard

      default:
        return pubsub.NAckDiscard
    }
  }
}

func HandlerWar(gs *gamelogic.GameState) func(gamelogic.RecognitionOfWar) pubsub.AckType {
    return func(war gamelogic.RecognitionOfWar) pubsub.AckType {
      defer fmt.Print("> ")
      outcome, _, _ := gs.HandleWar(war) 
      switch outcome {
        case gamelogic.WarOutcomeNotInvolved:
            return pubsub.NackRequeue

        case gamelogic.WarOutcomeNoUnits:
            return pubsub.NAckDiscard

        case gamelogic.WarOutcomeOpponentWon,
             gamelogic.WarOutcomeYouWon,
             gamelogic.WarOutcomeDraw:
            return pubsub.Ack

        default:
            fmt.Printf("ERROR: Unexpected war outcome: %v\n", outcome)
            return pubsub.NAckDiscard
        }
    }
}
