package service

type Service struct {
	messenger Messenger
}

func New(messenger Messenger) *Service {
	return &Service{
		messenger: messenger,
	}
}
