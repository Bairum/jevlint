class EmployeeDirectory {
	string[] names = new string[64];
	string[] departments = new string[64];

	void Add(int index, string name, string department) {
		names[index] = name;
		departments[index] = department;
	}
}
