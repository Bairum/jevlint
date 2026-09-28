int load_config(char *buffer, int size);

int load_settings(char *buffer, int size) {
	int result = load_config(buffer, size);
	if (result != 0) {
		return result;
	}
	return 0;
}
