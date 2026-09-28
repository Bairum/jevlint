class Formatter {
	String format(String value, boolean flag) {
		if (flag) {
			return "json";
		}
		return "xml";
	}
}
