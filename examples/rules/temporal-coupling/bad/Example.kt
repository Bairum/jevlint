class Example {
	class Rectangle {
		private var width = 0
		private var height = 0
		private var initialized = false

		fun init(width: Int, height: Int) {
			this.width = width
			this.height = height
			initialized = true
		}

		fun area(): Int {
			if (!initialized) return 0
			return width * height
		}
	}
}
