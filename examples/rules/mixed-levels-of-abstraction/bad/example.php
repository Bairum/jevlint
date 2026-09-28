<?php

function applyTax($amount) { return $amount + $amount * 8 / 100; }

function buildTotal($quantity, $unitPrice) {
	$subtotal = $quantity * $unitPrice;
	$discount = $subtotal / 10;
	$taxed = applyTax($subtotal - $discount);
	return $taxed + 250;
}
