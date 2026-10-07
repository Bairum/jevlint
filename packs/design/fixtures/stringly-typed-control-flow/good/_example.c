enum Command { CMD_START, CMD_STOP };

int run_command(enum Command command) {
	switch (command) {
	case CMD_START:
		return 1;
	case CMD_STOP:
		return 0;
	}
	return -1;
}
