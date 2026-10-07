class Config:
    def __init__(self, path):
        self.path = path
        self.handle = open(path)
