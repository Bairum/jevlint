class Config {
public:
	Config(const char *path) {
		handle = fopen(path, "r");
	}
private:
	void *handle;
};
