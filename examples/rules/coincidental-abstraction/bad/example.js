function isOverBudget(amount) {
	return amount > 1000;
}

function checkSystems(balance, cpuLoad) {
	const overBudget = isOverBudget(balance);
	const overloaded = isOverBudget(cpuLoad);
	return overBudget || overloaded;
}
