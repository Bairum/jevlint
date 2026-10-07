struct FormatOptions {
    int uppercase;
    int padding;
    int verbose;
};

int format_count(int count, struct FormatOptions options) {
    return count + 1;
}
