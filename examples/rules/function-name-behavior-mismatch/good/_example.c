struct User {
	char *name;
};

char *get_name(struct User *user) {
	return user->name;
}
