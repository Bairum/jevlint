<?php

class User {
	public string $name = "";
	public int $reads = 0;

	public function getName(): string {
		$this->reads++;
		return $this->name;
	}
}
