<?php

class Rectangle {
	public function __construct(private int $width, private int $height) {
	}

	public function area(): int {
		return $this->width * $this->height;
	}
}
