class FormatOptions(
    val uppercase: Boolean = false,
    val padding: Int = 0,
    val verbose: Boolean = false,
)

class Formatter {
    fun format(name: String, options: FormatOptions = FormatOptions()): String {
        return name
    }
}
