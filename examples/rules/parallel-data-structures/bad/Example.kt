class EmployeeDirectory {
	val names = arrayOfNulls<String>(64)
	val departments = arrayOfNulls<String>(64)

	fun add(index: Int, name: String, department: String) {
		names[index] = name
		departments[index] = department
	}
}
