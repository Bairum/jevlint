<?php

function format(string $value, bool $flag): string {
	if ($flag) {
		return "json";
	}
	return "xml";
}
