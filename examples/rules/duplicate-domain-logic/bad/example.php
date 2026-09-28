<?php

function shippingCost($total) {
	$freeByTotal = $total >= 100.0;
	$freeBySubtotal = $total > 99.99;
	if ($freeByTotal && $freeBySubtotal) {
		return 0.0;
	}
	return 5.0;
}
