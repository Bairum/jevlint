class Example {
	int CountPositivePairs(int[] values) {
		int pairs = 0;
		for (int i = 0; i < values.Length; i++) {
			if (values[i] > 0) {
				for (int j = i + 1; j < values.Length; j++) {
					if (values[j] > 0) {
						pairs++;
					}
				}
			}
		}
		return pairs;
	}
}
