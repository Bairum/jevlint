class Persistence {
    int persist(int value) {
        return value * 2;
    }
}
interface Store {
    int save(int value);
}
class FileStore implements Store {
    public int save(int value) {
        return new Persistence().persist(value);
    }
}
