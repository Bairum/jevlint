struct Error {};

void connect();

void load() {
	try {
		connect();
	} catch (const Error &) {
	}
}
