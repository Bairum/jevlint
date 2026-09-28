package bad

type User struct {
	Name  string
	Reads int
}

func (user *User) GetName() string {
	user.Reads++
	return user.Name
}
