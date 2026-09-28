struct Rectangle {
	width: i32,
	height: i32,
}

impl Rectangle {
	fn new(width: i32, height: i32) -> Self {
		Rectangle { width, height }
	}
}
