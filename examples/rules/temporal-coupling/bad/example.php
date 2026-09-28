<?php

class Rectangle {
	private int $width;
	private int $height;
	private bool $initialized = false;

	public function init(int $width, int $height): void {
		$this->width = $width;
		$this->height = $height;
		$this->initialized = true;
	}

	public function area(): int {
		if (!$this->initialized) return 0;
		return $this->width * $this->height;
	}
}
