package good

type LoadState int

const (
	Idle LoadState = iota
	Loading
	Loaded
)
