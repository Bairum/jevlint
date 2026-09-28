int run_command(const char *command) {
	if (strcmp(command, "start") == 0) {
		return 1;
	}
	if (strcmp(command, "stop") == 0) {
		return 0;
	}
	return -1;
}
