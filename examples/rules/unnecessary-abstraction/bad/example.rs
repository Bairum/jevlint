fn persist(value: i32) -> i32 {
    value * 2
}
trait Store {
    fn save(&self, value: i32) -> i32;
}
struct FileStore;
impl Store for FileStore {
    fn save(&self, value: i32) -> i32 {
        persist(value)
    }
}
