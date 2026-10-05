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
