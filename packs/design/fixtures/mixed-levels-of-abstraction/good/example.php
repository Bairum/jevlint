<?php

function calculateSubtotal($quantity, $unitPrice) { return $quantity * $unitPrice; }

function applyTax($amount) { return $amount + $amount * 8 / 100; }

function buildTotal($quantity, $unitPrice) {
	$subtotal = calculateSubtotal($quantity, $unitPrice);
	return applyTax($subtotal);
}
