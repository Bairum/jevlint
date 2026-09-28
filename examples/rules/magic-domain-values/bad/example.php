<?php

function can_retry($attempts) {
	return $attempts < 3;
}
