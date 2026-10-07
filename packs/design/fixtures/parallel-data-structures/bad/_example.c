char *employee_names[64];
char *employee_departments[64];

void add_employee(int index, char *name, char *department) {
	employee_names[index] = name;
	employee_departments[index] = department;
}
