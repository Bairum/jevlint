class Rectangle {
public:
	Rectangle() : initialized_(false) {}

	void Init(int width, int height) {
		width_ = width;
		height_ = height;
		initialized_ = true;
	}

	int Area() const {
		if (!initialized_) return 0;
		return width_ * height_;
	}

private:
	int width_;
	int height_;
	bool initialized_;
};
