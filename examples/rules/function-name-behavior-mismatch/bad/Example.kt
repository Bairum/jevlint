class Example {
	private val name: String = ""
	private var reads: Int = 0

	fun getName(): String {
		reads++
		return name
	}
}
