typedef int *(*QueryFn)(const char *);

int *load_team_links(QueryFn query) {
	int *userTeamIds = query("SELECT team_id FROM users");
	int *teamIds = query("SELECT id FROM teams");
	for (int i = 0; i < 10; i++) {
		for (int j = 0; j < 10; j++) {
			if (userTeamIds[i] == teamIds[j]) {
				userTeamIds[i] = teamIds[j];
			}
		}
	}
	return userTeamIds;
}
