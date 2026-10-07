class Rectangle {
	private width = 0;
	private height = 0;
	private initialized = false;

	init(width: number, height: number): void {
		this.width = width;
		this.height = height;
		this.initialized = true;
	}

	area(): number {
		if (!this.initialized) return 0;
		return this.width * this.height;
	}
}
