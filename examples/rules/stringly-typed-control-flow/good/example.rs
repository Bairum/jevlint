enum Command {
	Start,
	Stop,
}

fn run(command: Command) -> i32 {
	match command {
		Command::Start => 1,
		Command::Stop => 0,
	}
}
