class Rectangle {
public:
	Rectangle(int width, int height) : width_(width), height_(height) {}

	int Area() const {
		return width_ * height_;
	}

private:
	int width_;
	int height_;
};
