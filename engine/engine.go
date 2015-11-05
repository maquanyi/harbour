package engine

type Engine struct {
}

var (
	DockerSock       string
	SocketGroup      string
	ContainerRuntime string
)

func New() *Engine {
	eng := &Engine{}
	return eng
}
