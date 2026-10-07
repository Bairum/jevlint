class CommandRunner {
	int run(String command) {
		if (command.equals("start")) {
			return 1;
		}
		if (command.equals("stop")) {
			return 0;
		}
		return -1;
	}
}
