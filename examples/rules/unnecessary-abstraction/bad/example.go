package bad

type Store interface {
	Save(int) int
}

type FileStore struct{}

func Persist(value int) int {
	return value * 2
}

func (FileStore) Save(value int) int {
	return Persist(value)
}
