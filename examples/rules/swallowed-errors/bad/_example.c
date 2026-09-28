int load_config(char *buffer, int size);

int load_settings(char *buffer, int size) {
	load_config(buffer, size);
	return 0;
}
