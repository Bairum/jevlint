function persist(value) {
  return value * 2;
}
class Store {
  save(value) {
    return persist(value);
  }
}
