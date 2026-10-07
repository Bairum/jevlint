typedef int *(*QueryFn)(const char *);

int *load_team_links(QueryFn query) {
	return query("SELECT users.team_id FROM users JOIN teams ON teams.id = users.team_id");
}
