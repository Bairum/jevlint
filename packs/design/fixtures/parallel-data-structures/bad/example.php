<?php

$employeeNames = [];
$employeeDepartments = [];

function add_employee($index, $name, $department) {
	global $employeeNames, $employeeDepartments;
	$employeeNames[$index] = $name;
	$employeeDepartments[$index] = $department;
}
