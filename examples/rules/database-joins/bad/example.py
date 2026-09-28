def load_team_links(database):
    user_team_ids = database.query("SELECT team_id FROM users")
    team_ids = database.query("SELECT id FROM teams")
    joined = []
    for user_id in user_team_ids:
        for team_id in team_ids:
            if user_id == team_id:
                joined.append(user_id)
    return joined
