package bad

type Rectangle struct {
	Width       int
	Height      int
	initialized bool
}

func (r *Rectangle) Init(width int, height int) {
	r.Width = width
	r.Height = height
	r.initialized = true
}

func (r Rectangle) Area() int {
	if !r.initialized {
		return 0
	}
	return r.Width * r.Height
}
