interface Database {
	int[] Query(string sql);
}

class Example {
	int[] LoadTeamLinks(Database database) {
		return database.Query("SELECT users.team_id FROM users JOIN teams ON teams.id = users.team_id");
	}
}
