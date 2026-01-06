package main

import (
	"fmt"
	"learn-pub-sub-starter/internal/gamelogic"
	"learn-pub-sub-starter/internal/routing"
)


func HandlerPause(gs *gamelogic.GameState) func(routing.PlayingState){
  defer fmt.Print("> ")
  return gs.HandlePause
}

func HandlerMove(gs *gamelogic.GameState) func(gamelogic.ArmyMove) {
  defer fmt.Print("\n> ")
  return func(move gamelogic.ArmyMove) {
    gs.HandleMove(move)
  }
}
