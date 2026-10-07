class LoadStatus {
	enum class State { IDLE, LOADING, LOADED }
	val state: State = State.IDLE
}
