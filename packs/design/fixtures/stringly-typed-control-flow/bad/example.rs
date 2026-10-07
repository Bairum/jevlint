fn run(command: &str) -> i32 {
	if command == "start" {
		return 1;
	}
	if command == "stop" {
		return 0;
	}
	-1
}
