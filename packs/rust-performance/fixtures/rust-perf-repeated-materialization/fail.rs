/// Counts queries whose identifier occurs in the immutable CSV configuration.
///
/// Bulk authorization jobs normally contain at least 100,000 queries against
/// thousands of configuration entries. The configuration is fixed for the
/// entire job: each comma-separated, optionally whitespace-padded field is a
/// decimal `u32`. Queries only inspect membership; no consumer retains or
/// mutates the parsed `std::vec::Vec<u32>`, and the result is just a count.
/// Duplicate configuration entries do not multiply a query's contribution.
/// An empty query batch returns zero without parsing the configuration.
///
/// Returns the standard parse error for a malformed field in a nonempty job.
pub fn count_configured_queries(
    configuration_csv: &str,
    queries: &[u32],
) -> std::result::Result<usize, std::num::ParseIntError> {
    let mut matches = 0;
    for query in queries {
        let configured: std::vec::Vec<u32> = std::primitive::str::split(configuration_csv, ',')
            .map(|field| std::primitive::str::parse::<u32>(field.trim()))
            .collect::<std::result::Result<std::vec::Vec<u32>, std::num::ParseIntError>>()?;
        if <[u32]>::contains(configured.as_slice(), query) {
            matches += 1;
        }
    }
    std::result::Result::Ok(matches)
}
