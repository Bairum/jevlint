package bad

type User struct {
	Name   string
	TeamID int
}

type Team struct {
	ID   int
	Name string
}

type UserWithTeam struct {
	UserName string
	TeamName string
}

type Database interface {
	Query(string) []map[string]string
}

func LoadUsersWithTeams(database Database) []UserWithTeam {
	users := database.Query("SELECT name, team_id FROM users")
	teams := database.Query("SELECT id, name FROM teams")
	joined := make([]UserWithTeam, 0, len(users))
	for _, user := range users {
		for _, team := range teams {
			if team["id"] == user["team_id"] {
				joined = append(joined, UserWithTeam{
					UserName: user["name"],
					TeamName: team["name"],
				})
			}
		}
	}
	return joined
}
