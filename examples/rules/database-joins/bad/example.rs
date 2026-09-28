trait Database {
	fn query(&self, sql: &str) -> Vec<i32>;
}

fn load_team_links(database: &dyn Database) -> Vec<i32> {
	let user_team_ids = database.query("SELECT team_id FROM users");
	let team_ids = database.query("SELECT id FROM teams");
	let mut joined = Vec::new();
	for user_id in &user_team_ids {
		for team_id in &team_ids {
			if user_id == team_id {
				joined.push(*user_id);
			}
		}
	}
	joined
}
