package good

type User struct {
	Name string
}

func (user *User) GetName() string {
	return user.Name
}
