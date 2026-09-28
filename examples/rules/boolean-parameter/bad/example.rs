fn format(value: &str, flag: bool) -> &str {
	if flag {
		return "json";
	}
	"xml"
}
