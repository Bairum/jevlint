<?php

function loadTeamLinks($database) {
	$userTeamIds = $database->query("SELECT team_id FROM users");
	$teamIds = $database->query("SELECT id FROM teams");
	$joined = [];
	foreach ($userTeamIds as $userId) {
		foreach ($teamIds as $teamId) {
			if ($userId === $teamId) {
				$joined[] = $userId;
			}
		}
	}
	return $joined;
}
