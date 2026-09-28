def count_positive_pairs(values):
    pairs = 0
    for i, first in enumerate(values):
        if first > 0:
            for second in values[i + 1:]:
                if second > 0:
                    pairs += 1
    return pairs
