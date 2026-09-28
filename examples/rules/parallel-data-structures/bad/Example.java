class EmployeeDirectory {
	String[] names = new String[64];
	String[] departments = new String[64];

	void add(int index, String name, String department) {
		names[index] = name;
		departments[index] = department;
	}
}
