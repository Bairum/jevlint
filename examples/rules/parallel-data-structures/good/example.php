<?php

class Employee {
	public $name;
	public $department;

	public function __construct($name, $department) {
		$this->name = $name;
		$this->department = $department;
	}
}
