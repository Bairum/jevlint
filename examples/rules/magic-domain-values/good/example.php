<?php

const MAX_RETRY_ATTEMPTS = 3;

function can_retry($attempts) {
	return $attempts < MAX_RETRY_ATTEMPTS;
}
