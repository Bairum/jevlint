package good

func Load() error {
	err := connect()
	if err != nil {
		return err
	}
	return nil
}

func connect() error {
	return nil
}
