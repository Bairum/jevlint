<?php

function run(string $command): int {
	if ($command === "start") {
		return 1;
	}
	if ($command === "stop") {
		return 0;
	}
	return -1;
}
