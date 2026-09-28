<?php

class AppOptions {
	public bool $enableCache = false;
	public bool $enableLogging = false;
	public bool $sendEmail = false;
	public bool $darkMode = false;
	public int $retryCount = 0;
	public string $databaseUrl = "";
	public int $maxConnections = 0;
	public string $smtpHost = "";
}
