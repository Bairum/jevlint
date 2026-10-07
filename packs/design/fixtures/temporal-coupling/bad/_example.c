typedef struct {
	int width;
	int height;
	int initialized;
} Rectangle;

void rectangle_init(Rectangle *rectangle, int width, int height) {
	rectangle->width = width;
	rectangle->height = height;
	rectangle->initialized = 1;
}

int rectangle_area(Rectangle *rectangle) {
	if (!rectangle->initialized) return 0;
	return rectangle->width * rectangle->height;
}
