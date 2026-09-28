class Example {
	int countPositivePairs(int[] values) {
		int pairs = 0;
		for (int i = 0; i < values.length; i++) {
			if (values[i] > 0) {
				for (int j = i + 1; j < values.length; j++) {
					if (values[j] > 0) {
						pairs++;
					}
				}
			}
		}
		return pairs;
	}
}
