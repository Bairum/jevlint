def load_team_links(database)
  user_team_ids = database.query("SELECT team_id FROM users")
  team_ids = database.query("SELECT id FROM teams")
  joined = []
  user_team_ids.each do |user_id|
    team_ids.each do |team_id|
      joined << user_id if user_id == team_id
    end
  end
  joined
end
