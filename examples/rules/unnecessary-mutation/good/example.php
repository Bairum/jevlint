<?php

function computeTotal($quantity) {
	$listPrice = $quantity * 12.5;
	$discounted = $listPrice * 0.9;
	$total = $discounted + 4.0;
	return $total;
}
