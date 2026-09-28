class CommandRunner {
	enum class Command { START, STOP }

	fun run(command: Command): Int = when (command) {
		Command.START -> 1
		Command.STOP -> 0
	}
}
