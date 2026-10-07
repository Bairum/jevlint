typedef struct {
	int width;
	int height;
} Rectangle;

Rectangle rectangle_new(int width, int height) {
	Rectangle rectangle;
	rectangle.width = width;
	rectangle.height = height;
	return rectangle;
}
