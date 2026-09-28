<?php

function computeTotal($quantity) {
	$value = $quantity;
	$value = $value * 12.5;
	$value = $value * 0.9;
	$value = $value + 4.0;
	return $value;
}
