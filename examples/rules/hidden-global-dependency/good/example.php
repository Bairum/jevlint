<?php

function calculateTax(int $amount, int $taxRate): int {
	return intdiv($amount * $taxRate, 100);
}
