class User:
    def __init__(self, name):
        self.name = name
        self.reads = 0

    def get_name(self):
        self.reads += 1
        return self.name
