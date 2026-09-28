package bad

var employeeNames []string
var employeeDepartments []string

func AddEmployee(index int, name string, department string) {
	employeeNames[index] = name
	employeeDepartments[index] = department
}
