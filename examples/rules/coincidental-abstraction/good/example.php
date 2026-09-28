<?php

function isOverBudget($balance) {
	return $balance > 1000.0;
}

function isOverloaded($cpuLoad) {
	return $cpuLoad > 80.0;
}
