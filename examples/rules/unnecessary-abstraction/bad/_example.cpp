int persist(int value) {
    return value * 2;
}

class Store {
public:
    int save(int value) {
        return persist(value);
    }
};
