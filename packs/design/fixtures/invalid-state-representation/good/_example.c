typedef enum { IDLE, LOADING, LOADED } LoadState;

struct LoadStatus {
	LoadState state;
};
