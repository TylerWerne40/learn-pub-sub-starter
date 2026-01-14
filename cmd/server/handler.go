package main

import (
	"fmt"
	"learn-pub-sub-starter/internal/gamelogic"
	"learn-pub-sub-starter/internal/pubsub"
	"learn-pub-sub-starter/internal/routing"
)

func log_handler(log routing.GameLog) pubsub.AckType {
  defer fmt.Print("\n >")
  gamelogic.WriteLog(log)
  return pubsub.Ack
}
