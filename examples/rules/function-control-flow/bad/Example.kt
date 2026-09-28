class Example {
	fun countPositivePairs(values: IntArray): Int {
		var pairs = 0
		for (i in values.indices) {
			if (values[i] > 0) {
				for (j in i + 1 until values.size) {
					if (values[j] > 0) {
						pairs++
					}
				}
			}
		}
		return pairs
	}
}
