package gamelogic

import (
	"context"
	"fmt"

	"learn-pub-sub-starter/internal/pubsub"
	"learn-pub-sub-starter/internal/routing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishGameLog(ctx context.Context, ch *amqp.Channel, log routing.GameLog) error {
  if log.Username == "" {
    return fmt.Errorf("cannot publish GameLog: empty username in game state")
  }

  routingKey := routing.GameLogSlug + "." + log.Username

  return pubsub.PublishGob(ctx, ch, routing.ExchangePerilTopic, routingKey, log)
}
