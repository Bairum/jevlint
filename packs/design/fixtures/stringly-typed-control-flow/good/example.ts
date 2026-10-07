enum Command {
	Start,
	Stop,
}

function run(command: Command): number {
	switch (command) {
		case Command.Start:
			return 1;
		case Command.Stop:
			return 0;
	}
	return -1;
}
