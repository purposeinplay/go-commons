package kafka

import (
	"log"
	"os"

	"github.com/IBM/sarama"
)

func init() {
	sarama.Logger = log.New(os.Stdout, "[Sarama] ", log.LstdFlags)
}
