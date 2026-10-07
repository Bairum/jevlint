package good

type Rectangle struct {
	Width  int
	Height int
}

func NewRectangle(width int, height int) Rectangle {
	return Rectangle{Width: width, Height: height}
}
