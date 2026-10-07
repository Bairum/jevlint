class Example {
	fun maxValue(a: Int, b: Int): Int {
		if (a > b) {
			return a
		} else {
			if (b > a) {
				return b
			} else {
				return a
			}
		}
	}
}
