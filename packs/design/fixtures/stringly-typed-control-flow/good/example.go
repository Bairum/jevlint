package good

type Command int

const (
	Start Command = iota
	Stop
)

func Run(command Command) int {
	switch command {
	case Start:
		return 1
	case Stop:
		return 0
	}
	return -1
}
