function countPositivePairs(values: number[]): number {
	let pairs = 0;
	for (let i = 0; i < values.length; i++) {
		if (values[i] > 0) {
			for (let j = i + 1; j < values.length; j++) {
				if (values[j] > 0) {
					pairs += 1;
				}
			}
		}
	}
	return pairs;
}
