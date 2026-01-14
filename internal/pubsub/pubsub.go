package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
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


func SubscribeGob[T any] (
  conn *amqp.Connection,
  exchange,
  queueName,
  key string,
  queueType SimpleQueueType,
  handler func(T) AckType,
) error {
  unmarshaller := func (data []byte) (T, error) {
	  var ret T
	  b := bytes.NewBuffer(data)
	  decoder := gob.NewDecoder(b)
	  err := decoder.Decode(&ret)
	  if err != nil {
		  return ret, err
	  }
	  return ret, nil
  } 
  err := subscribe(
    conn,
    exchange,
    queueName,
    key,
    queueType,
    handler,
    unmarshaller,
    )
  if err != nil {
    return fmt.Errorf("Could not Subscribe using Gob, %v\n", err)
  }
  return nil
}

func subscribe[T any] (
  conn *amqp.Connection,
  exchange,
  queueName,
  key string,
  queueType SimpleQueueType,
  handler func(T) AckType,
  unmarshaller func([]byte) (T, error),
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
    
    defer chann.Close()
    for msg := range msgs {
      var payload T
      payload, err = unmarshaller(msg.Body)
      if err != nil {
        fmt.Printf("Failed to Unmarshall message (%T): %v\n", payload, err)
        msg.Nack(false, true)
        continue
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

func SubscribeJSON[T any] (
  conn *amqp.Connection,
  exchange,
  queueName,
  key string,
  queueType SimpleQueueType,
  handler func(T) AckType,
) error {
  unmarshaller := func (data []byte) (T, error) {
    var payload T
    err := json.Unmarshal(data, &payload)
    if err != nil {
      return payload, err
    }
    return payload, nil 
  }
  err := subscribe(
    conn,
    exchange,
    queueName,
    key,
    queueType,
    handler,
    unmarshaller,
    )
  if err != nil {
    return fmt.Errorf("Could not subscribe to JSON: %v\n", err)
  }
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

func PublishGob[T any](ctx context.Context, ch *amqp.Channel, exchange, key string, val T) error {
  var b bytes.Buffer
  enc := gob.NewEncoder(&b)
  err := enc.Encode(val)
  if err != nil {
    return err
  }
  return ch.PublishWithContext(
    ctx,
    exchange,
    key,
    false,
    false,
  amqp.Publishing{
    ContentType: "application/gob",
    Body: b.Bytes(),
    },
    )
}

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
