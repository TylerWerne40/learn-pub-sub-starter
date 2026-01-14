package main

import (
	"context"
	"fmt"
	"learn-pub-sub-starter/internal/gamelogic"
	"learn-pub-sub-starter/internal/pubsub"
	"learn-pub-sub-starter/internal/routing"
	"time"

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

        return pubsub.Ack

      case gamelogic.MoveOutComeSafe:
        return pubsub.Ack
      case gamelogic.MoveOutcomeSamePlayer:
        return pubsub.NAckDiscard

      default:
        return pubsub.NAckDiscard
    }
  }
}

func HandlerWar(ctx context.Context, gs *gamelogic.GameState, chann *amqp.Channel) func(gamelogic.RecognitionOfWar) pubsub.AckType {
    return func(war gamelogic.RecognitionOfWar) pubsub.AckType {
      defer fmt.Print("> ")
      outcome, winner, loser := gs.HandleWar(war) 
      gameLog := routing.GameLog{
        Username:     gs.GetUsername(),
        CurrentTime:  time.Now(),
        Message:      "",
      }
      switch outcome {
        case gamelogic.WarOutcomeNotInvolved:
          gameLog.Message = "Not Involved"

        case gamelogic.WarOutcomeNoUnits: 
          gameLog.Message = "No Units"

        case gamelogic.WarOutcomeOpponentWon, 
            gamelogic.WarOutcomeYouWon:
          gameLog.Message = winner + " won a war against " + loser
          fmt.Println(winner, " won a war against ", loser)
        case gamelogic.WarOutcomeDraw:
            gameLog.Message = "A war between " + winner + " and " + loser + " resulted in a draw"
            fmt.Println("A war between ", winner, " and ", loser, " resulted in a draw")

        default:
          fmt.Printf("ERROR: Unexpected war outcome: %v\n", outcome)
          gameLog.Message = fmt.Sprintf("ERROR: Unexpected war outcome:%v", outcome)
        }
      err := publishGameLog(ctx, chann, gameLog.Username, gameLog.Message)
      if err != nil {
        fmt.Println("Failed to publish game log:", err)
        return pubsub.NAckDiscard
      }
      return pubsub.Ack
    }
}


func publishGameLog(ctx context.Context, ch *amqp.Channel, username, msg string) error {
	gameLog := routing.GameLog{
		Username:    username,
		CurrentTime: time.Now(),
		Message:     msg,
	}
  return gamelogic.PublishGameLog(ctx, ch, gameLog)
}
