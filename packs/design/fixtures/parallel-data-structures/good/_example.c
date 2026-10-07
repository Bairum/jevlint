struct Employee {
	char *name;
	char *department;
};

char *employee_department(struct Employee *employee) {
	return employee->department;
}
