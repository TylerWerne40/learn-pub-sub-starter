package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"learn-pub-sub-starter/internal/routing"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AckType int

const (
  Ack AckType = iota
  NackRequeue
  NAckDiscard
)

func SubscribeJSON[T any] (
  conn *amqp.Connection,
  exchange,
  queueName,
  key string,
  queueType SimpleQueueType,
  handler func(T) AckType,
) error {
  chann, queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
  if err != nil {
    return err
  }
  msgs, err := chann.Consume( queue.Name, "", false, false, false, false, nil)
  if err != nil {
    return err
  }
  go func() {
    for msg := range msgs {
      var payload T
      err = json.Unmarshal(msg.Body, &payload)
      if err != nil {
        fmt.Println("Failed to Unmarshall message: ", err)
        msg.Nack(false, true)
        return
      }
    
      var ack AckType = handler(payload)
      switch ack {
        case Ack:
          msg.Ack(false)
          fmt.Println("Acknowledged!")
        case NackRequeue:
          msg.Nack(false, true)
          fmt.Println("Not Acknowledged! Requeue")
        case NAckDiscard:
          msg.Nack(false, false)
          fmt.Println("Not Acknowledged! Discard")
      }
    }
  }()
  return nil
}
func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
  bytes, err := json.Marshal(val)
  if err != nil {
    return err
  }
  var pub amqp.Publishing
  pub.ContentType = "application/json"
  pub.Body = bytes
  err = ch.PublishWithContext(context.Background(), exchange, key, false, false, pub)
  return err
}

type SimpleQueueType string

const (
  Durable SimpleQueueType = "durable"
  Transient SimpleQueueType = "transient"
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {
  chann, err := conn.Channel()
  if err != nil {
    return nil, amqp.Queue{}, err // figure out amqp.Queue declaration
  }
  var queue amqp.Queue
  args := amqp.Table{
    "x-dead-letter-exchange": routing.ExchangePerilDlx,
  }
  switch queueType {
  case Transient:
    queue, err := chann.QueueDeclare(queueName, false, true, true, false, args)
    
    if err != nil {
      return nil, queue, err
    }
  case Durable:
    queue, err := chann.QueueDeclare(queueName, true, false, false, false, args) 
    
    if err != nil {
      return nil, queue, err
    }
  }
  err = chann.QueueBind(queueName, key, exchange, false, nil)
  if err != nil {
    return nil, queue, err
  }
  return chann, queue, nil
}
