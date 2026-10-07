interface Database {
	int[] query(String sql);
}

class Example {
	int[] loadTeamLinks(Database database) {
		int[] userTeamIds = database.query("SELECT team_id FROM users");
		int[] teamIds = database.query("SELECT id FROM teams");
		for (int i = 0; i < userTeamIds.length; i++) {
			for (int j = 0; j < teamIds.length; j++) {
				if (userTeamIds[i] == teamIds[j]) {
					userTeamIds[i] = teamIds[j];
				}
			}
		}
		return userTeamIds;
	}
}
