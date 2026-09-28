class Example {
	fun load() {
		try {
			connect()
		} catch (error: Exception) {
			println(error.message)
		}
	}

	fun connect() {
	}
}
