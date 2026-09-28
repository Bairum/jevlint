class Example {
	fun clamp(value: Int, low: Int, high: Int): Int {
		if (value < low) {
			return low
		}
		if (value > high) {
			return high
		}
		return value
	}
}
