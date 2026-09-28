interface Database {
	fun query(sql: String): IntArray
}

class Example {
	fun loadTeamLinks(database: Database): IntArray {
		return database.query("SELECT users.team_id FROM users JOIN teams ON teams.id = users.team_id")
	}
}
