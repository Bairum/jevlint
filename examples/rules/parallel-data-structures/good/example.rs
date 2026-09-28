struct Employee {
	name: String,
	department: String,
}

impl Employee {
	fn department(&self) -> &str {
		&self.department
	}
}
