struct Rectangle {
	width: i32,
	height: i32,
	initialized: bool,
}

impl Rectangle {
	fn init(&mut self, width: i32, height: i32) {
		self.width = width;
		self.height = height;
		self.initialized = true;
	}

	fn area(&self) -> i32 {
		if !self.initialized {
			return 0;
		}
		self.width * self.height
	}
}
