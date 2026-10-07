<?php

$taxRate = 20;

function calculateTax(int $amount): int {
	global $taxRate;
	return intdiv($amount * $taxRate, 100);
}
