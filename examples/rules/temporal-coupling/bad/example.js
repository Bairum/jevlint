class Rectangle {
	constructor() {
		this.initialized = false;
	}

	init(width, height) {
		this.width = width;
		this.height = height;
		this.initialized = true;
	}

	area() {
		if (!this.initialized) return 0;
		return this.width * this.height;
	}
}
