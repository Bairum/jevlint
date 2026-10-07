function loadTeamLinks(database) {
	return database.query("SELECT users.team_id FROM users JOIN teams ON teams.id = users.team_id");
}
