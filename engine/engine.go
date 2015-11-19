package engine

type Engine struct {
	RuntimeType int
}

var (
	DockerSock  string
	SocketGroup string
)

const (
	RuntimeDocker = iota
	RuntimeRkt
)

func New(RuntimeType int) *Engine {
	eng := &Engine{}
	eng.RuntimeType = RuntimeType

	return eng
}
