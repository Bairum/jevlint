<?php

class Config {
	public $handle;

	public function __construct($path) {
		$this->handle = fopen($path, 'r');
	}
}
