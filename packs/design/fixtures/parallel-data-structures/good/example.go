package good

type Employee struct {
	Name       string
	Department string
}

func (employee Employee) DepartmentName() string {
	return employee.Department
}
