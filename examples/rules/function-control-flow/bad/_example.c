int count_positive_pairs(int *values, int count) {
	int pairs = 0;
	for (int i = 0; i < count; i++) {
		if (values[i] > 0) {
			for (int j = i + 1; j < count; j++) {
				if (values[j] > 0) {
					pairs++;
				}
			}
		}
	}
	return pairs;
}
