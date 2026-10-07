const Command = { Start: 1, Stop: 2 };

function run(command) {
	switch (command) {
		case Command.Start:
			return 1;
		case Command.Stop:
			return 0;
	}
	return -1;
}
