class Config(val path: String, val data: String)

fun createConfig(path: String): Config {
	val data = java.io.File(path).readText()
	return Config(path, data)
}
