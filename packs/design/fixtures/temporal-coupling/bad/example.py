class Rectangle:
    def __init__(self):
        self.initialized = False

    def init(self, width, height):
        self.width = width
        self.height = height
        self.initialized = True

    def area(self):
        if not self.initialized:
            return 0
        return self.width * self.height
