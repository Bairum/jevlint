struct Employee {
	const char *name;
	const char *department;
};

const char *employee_department(const Employee &employee) {
	return employee.department;
}
