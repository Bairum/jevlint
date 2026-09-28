class User {
	name = "";
	reads = 0;

	getName(): string {
		this.reads++;
		return this.name;
	}
}
