def persist(value):
    return value * 2

class Store:
    def save(self, value):
        return persist(value)
