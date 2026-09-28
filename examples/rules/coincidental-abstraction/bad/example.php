<?php

function isOverBudget($amount) {
	return $amount > 1000.0;
}

function checkSystems($balance, $cpuLoad) {
	$overBudget = isOverBudget($balance);
	$overloaded = isOverBudget($cpuLoad);
	return $overBudget || $overloaded;
}
