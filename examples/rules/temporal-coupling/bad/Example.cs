class Rectangle {
	private int width;
	private int height;
	private bool initialized;

	public void Initialize(int width, int height) {
		this.width = width;
		this.height = height;
		initialized = true;
	}

	public int Area() {
		if (!initialized) return 0;
		return width * height;
	}
}
