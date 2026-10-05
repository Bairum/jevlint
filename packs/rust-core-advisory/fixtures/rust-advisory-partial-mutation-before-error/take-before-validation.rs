use std::collections::BTreeMap;
use std::num::ParseIntError;

pub fn rescale_prices(
    prices: &mut BTreeMap<String, u64>,
    multiplier: &str,
) -> Result<(), ParseIntError> {
    let previous = std::mem::take(prices);
    let factor: u64 = multiplier.parse()?;
    for (product, amount) in previous {
        prices.insert(product, amount.saturating_mul(factor));
    }
    Ok(())
}
