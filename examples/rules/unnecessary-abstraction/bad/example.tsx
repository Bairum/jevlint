function persist(value: number): number {
  return value * 2;
}
class Store {
  save(value: number): number {
    return persist(value);
  }
}
