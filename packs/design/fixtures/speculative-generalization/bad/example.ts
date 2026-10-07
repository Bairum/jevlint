interface FormatOptions {
  uppercase?: boolean;
  padding?: number;
  verbose?: boolean;
}
function formatName(name: string, options: FormatOptions = {}): string {
  return name.trim();
}
