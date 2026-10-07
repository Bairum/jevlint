interface Database {
	fun query(sql: String): IntArray
}

class Example {
	fun loadTeamLinks(database: Database): IntArray {
		val userTeamIds = database.query("SELECT team_id FROM users")
		val teamIds = database.query("SELECT id FROM teams")
		val joined = mutableListOf<Int>()
		for (userId in userTeamIds) {
			for (teamId in teamIds) {
				if (userId == teamId) {
					joined.add(userId)
				}
			}
		}
		return joined.toIntArray()
	}
}
