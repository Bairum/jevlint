interface Database {
	query(sql: string): number[];
}

function loadTeamLinks(database: Database): number[] {
	const userTeamIds = database.query("SELECT team_id FROM users");
	const teamIds = database.query("SELECT id FROM teams");
	const joined: number[] = [];
	for (let i = 0; i < userTeamIds.length; i++) {
		for (let j = 0; j < teamIds.length; j++) {
			if (userTeamIds[i] === teamIds[j]) {
				joined.push(userTeamIds[i]);
			}
		}
	}
	return joined;
}
