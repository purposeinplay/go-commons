package pubsub

type EventName string

type Events interface {
	Emit(job Job) error
}
