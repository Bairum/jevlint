fn count_positive_pairs(values: &[i32]) -> i32 {
	let mut pairs = 0;
	for i in 0..values.len() {
		if values[i] > 0 {
			for j in i + 1..values.len() {
				if values[j] > 0 {
					pairs += 1;
				}
			}
		}
	}
	pairs
}
