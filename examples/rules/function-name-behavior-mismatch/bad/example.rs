struct User {
	name: String,
	reads: u32,
}

fn get_name(user: &mut User) -> &String {
	user.reads += 1;
	&user.name
}
