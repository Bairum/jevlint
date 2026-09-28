class Formatter {
	fun format(value: String, flag: Boolean): String {
		if (flag) {
			return "json"
		}
		return "xml"
	}
}
