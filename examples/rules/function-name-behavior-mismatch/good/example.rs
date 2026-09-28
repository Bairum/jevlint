struct User {
	name: String,
}

fn get_name(user: &User) -> &String {
	&user.name
}
