enum class Command { Start, Stop };

int runCommand(Command command) {
	switch (command) {
	case Command::Start:
		return 1;
	case Command::Stop:
		return 0;
	}
	return -1;
}
