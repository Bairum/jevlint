function isOverBudget(amount: number): boolean {
	return amount > 1000;
}

function checkSystems(balance: number, cpuLoad: number): boolean {
	const overBudget = isOverBudget(balance);
	const overloaded = isOverBudget(cpuLoad);
	return overBudget || overloaded;
}
