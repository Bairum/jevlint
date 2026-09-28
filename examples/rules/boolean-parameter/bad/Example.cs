class Formatter {
	string Format(string value, bool flag) {
		if (flag) {
			return "json";
		}
		return "xml";
	}
}
