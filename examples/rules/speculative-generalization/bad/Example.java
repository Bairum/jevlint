class FormatterOptions {
    boolean uppercase;
    int padding;
    boolean verbose;
}
class Formatter {
    String format(String name, FormatterOptions options) {
        return name + "!";
    }
}
