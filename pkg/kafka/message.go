package kafka

type Message struct {
	Key   []byte
	Data  []byte
	Topic string
}
