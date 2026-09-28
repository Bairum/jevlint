interface Database {
	query(sql: string): number[];
}

function loadTeamLinks(database: Database): number[] {
	return database.query("SELECT users.team_id FROM users JOIN teams ON teams.id = users.team_id");
}
