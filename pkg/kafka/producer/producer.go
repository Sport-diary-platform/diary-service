package producer

import (
	"context"
	"diary-service/pkg/kafka"
	"errors"

	"github.com/twmb/franz-go/pkg/kgo"
)

type Producer struct {
	client *kgo.Client
	opts   []kgo.Opt
}

func New(seeds []string, opts ...Option) (*Producer, error) {
	if len(seeds) == 0 {
		return nil, errors.New("kafka/producer - New - no seed brokers")
	}

	p := &Producer{
		client: nil,
	}
	p.opts = append(p.opts, kgo.SeedBrokers(seeds...))

	for _, opt := range opts {
		opt(p)
	}
	cl, err := kgo.NewClient(p.opts...)
	if err != nil {
		return nil, errors.Join(errors.New("kafka/producer - New - NewClient"), err)
	}
	p.client = cl
	p.opts = nil
	return p, nil
}

func (p *Producer) Ping(ctx context.Context) error {
	if err := p.client.Ping(ctx); err != nil {
		return errors.Join(errors.New("kafka/producer - Ping"), err)
	}
	return nil
}

func (p *Producer) Close() {
	p.client.Close()
}

func (p *Producer) Produce(ctx context.Context, msg ...*kafka.Message) error {
	if len(msg) == 0 {
		return errors.New("kafka/producer - Produce - zero messages sent")
	}
	msgs := make([]*kgo.Record, len(msg))
	for i, v := range msg {
		if v == nil {
			return errors.New("kafka/producer - Produce - nil message")
		}
		msgs[i] = &kgo.Record{
			Key:   v.Key,
			Value: v.Data,
			Topic: v.Topic,
		}
	}

	res := p.client.ProduceSync(ctx, msgs...)
	var err error
	for _, v := range res {
		if v.Err != nil {
			err = errors.Join(v.Err, err)
		}
	}
	if err != nil {
		return errors.Join(errors.New("kafka/producer - Produce - ProduceSync"), err)
	}

	return nil
}

func (p *Producer) ProduceAsync(ctx context.Context, msg *kafka.Message, promise func(error)) error {
	if msg == nil {
		return errors.New("kafka/producer - ProduceAsync - nil message")
	}
	if promise == nil {
		promise = func(error) {}
	}
	p.client.Produce(ctx, &kgo.Record{
		Key:   msg.Key,
		Value: msg.Data,
		Topic: msg.Topic,
	}, func(r *kgo.Record, err error) {
		promise(err)
	})
	return nil
}
