class CommandRunner {
	enum Command { Start, Stop }

	int Run(Command command) {
		switch (command) {
			case Command.Start: return 1;
			case Command.Stop: return 0;
		}
		return -1;
	}
}
