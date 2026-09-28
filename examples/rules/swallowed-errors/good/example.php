<?php

function load(): void {
	try {
		connect();
	} catch (Throwable $error) {
		error_log($error->getMessage());
	}
}

function connect(): void {
}
