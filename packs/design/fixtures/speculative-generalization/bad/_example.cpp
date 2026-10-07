struct FormatOptions {
    bool uppercase;
    int padding;
    bool verbose;
};

int formatCount(int count, const FormatOptions &options) {
    return count + 1;
}
