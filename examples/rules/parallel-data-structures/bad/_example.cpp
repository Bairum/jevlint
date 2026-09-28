const char *employee_names[64];
const char *employee_departments[64];

void add_employee(int index, const char *name, const char *department) {
	employee_names[index] = name;
	employee_departments[index] = department;
}
