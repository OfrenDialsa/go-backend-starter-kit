package service

type NsqClient interface {
	Publish(topic string, body []byte) error
	Stop()
}
