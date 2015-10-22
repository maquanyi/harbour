package engine

type Engine struct {
}

var (
	DockerSock  string
	SocketGroup string
)

func New() *Engine {
	eng := &Engine{}
	return eng
}
