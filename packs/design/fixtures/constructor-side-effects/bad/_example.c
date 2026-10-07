struct Config {
	void *handle;
};

void config_load(struct Config *config, const char *path) {
	config->handle = fopen(path, "r");
}
