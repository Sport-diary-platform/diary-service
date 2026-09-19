package producer

import (
	"github.com/twmb/franz-go/pkg/kgo"
)

type Option func(*Producer)

func GzipCompression() Option {
	return func(p *Producer) {
		p.opts = append(p.opts, kgo.ProducerBatchCompression(kgo.GzipCompression()))
	}
}

func SnappyCompression() Option {
	return func(p *Producer) {
		p.opts = append(p.opts, kgo.ProducerBatchCompression(kgo.SnappyCompression()))
	}
}
func Lz4Compression() Option {
	return func(p *Producer) {
		p.opts = append(p.opts, kgo.ProducerBatchCompression(kgo.Lz4Compression()))
	}
}

func ZstdCompression() Option {
	return func(p *Producer) {
		p.opts = append(p.opts, kgo.ProducerBatchCompression(kgo.ZstdCompression()))
	}
}

// func Prometeus(namespace, port string) Option {
// 	return func(p *Producer) {
// 		kgo.WithHooks()
// 	}
// }
