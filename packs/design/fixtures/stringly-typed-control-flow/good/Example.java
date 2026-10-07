class CommandRunner {
	enum Command { START, STOP }

	int run(Command command) {
		switch (command) {
			case START: return 1;
			case STOP: return 0;
		}
		return -1;
	}
}
