struct User {
	char *name;
	int reads;
};

char *get_name(struct User *user) {
	user->reads++;
	return user->name;
}
