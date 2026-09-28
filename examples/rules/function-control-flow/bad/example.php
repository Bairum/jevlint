<?php

function countPositivePairs($values) {
	$pairs = 0;
	for ($i = 0; $i < count($values); $i++) {
		if ($values[$i] > 0) {
			for ($j = $i + 1; $j < count($values); $j++) {
				if ($values[$j] > 0) {
					$pairs++;
				}
			}
		}
	}
	return $pairs;
}
