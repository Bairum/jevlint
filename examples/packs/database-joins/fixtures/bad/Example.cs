interface Database {
	int[] Query(string sql);
}

class Example {
	int[] LoadTeamLinks(Database database) {
		int[] userTeamIds = database.Query("SELECT team_id FROM users");
		int[] teamIds = database.Query("SELECT id FROM teams");
		for (int i = 0; i < userTeamIds.Length; i++) {
			for (int j = 0; j < teamIds.Length; j++) {
				if (userTeamIds[i] == teamIds[j]) {
					userTeamIds[i] = teamIds[j];
				}
			}
		}
		return userTeamIds;
	}
}
