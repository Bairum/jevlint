class Config {
	data: unknown;

	constructor(url: string) {
		this.data = fetch(url);
	}
}
