package ws

func NewServer(host string) (*Server, error) {
	c := &Server{
		host: host,
	}
	return c, nil
}

type Server struct {
	host string
}
