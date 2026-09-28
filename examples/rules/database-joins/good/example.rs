trait Database {
	fn query(&self, sql: &str) -> Vec<i32>;
}

fn load_team_links(database: &dyn Database) -> Vec<i32> {
	database.query("SELECT users.team_id FROM users JOIN teams ON teams.id = users.team_id")
}
