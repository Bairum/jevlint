fun persist(value: Int): Int = value * 2
interface Store {
    fun save(value: Int): Int
}
class FileStore : Store {
    override fun save(value: Int): Int = persist(value)
}
