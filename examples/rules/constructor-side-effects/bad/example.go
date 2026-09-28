package bad

type Database interface {
	Connect(string) error
}

type Store struct{ Handle string }

func NewStore(database Database, dsn string) *Store {
	database.Connect(dsn)
	return &Store{Handle: dsn}
}
