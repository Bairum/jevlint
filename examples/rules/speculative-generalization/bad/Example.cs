class FormatOptions
{
    public bool Uppercase { get; set; }
    public int Padding { get; set; }
}
class Formatter
{
    public string Format(string name, FormatOptions options)
    {
        return name + "!";
    }
}
