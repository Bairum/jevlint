int persist(int value) {
    return value * 2;
}

struct Store {
    int (*save)(int);
};

int store(int value) {
    struct Store s = { persist };
    return s.save(value);
}
