<?php

enum Mode {
	case Start;
	case Stop;
}

function run(Mode $command): int {
	switch ($command) {
		case Mode::Start:
			return 1;
		case Mode::Stop:
			return 0;
	}
	return -1;
}
