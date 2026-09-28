function format(value: string, flag: boolean): string {
	if (flag) {
		return "json";
	}
	return "xml";
}
