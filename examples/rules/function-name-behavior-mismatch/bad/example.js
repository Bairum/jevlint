class User {
	name = "";
	reads = 0;

	getName() {
		this.reads++;
		return this.name;
	}
}
