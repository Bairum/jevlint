fn add_employee(names: &mut Vec<String>, departments: &mut Vec<String>, name: String, department: String) {
	names.push(name);
	departments.push(department);
}

fn all_match(names: &[String], departments: &[String]) -> bool {
	names.len() == departments.len()
}
