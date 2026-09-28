package bad

func Format(value string, flag bool) string {
	if flag {
		return "json"
	}
	return "xml"
}
