class Example {
	static class Rectangle {
		private int width;
		private int height;
		private boolean initialized;

		void init(int width, int height) {
			this.width = width;
			this.height = height;
			this.initialized = true;
		}

		int area() {
			if (!initialized) return 0;
			return width * height;
		}
	}
}
