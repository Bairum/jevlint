const char *format(const char *value, bool flag) {
	if (flag) {
		return "json";
	}
	return "xml";
}
