#[derive(Clone, Debug, PartialEq)]
pub struct InventoryItem {
    pub sku: String,
    pub quantity: u32,
    pub reorder_at: u32,
}

pub struct InventoryView {
    pub warehouse: String,
    pub items: std::sync::Arc<std::sync::Mutex<Vec<InventoryItem>>>,
}

impl InventoryView {
    pub fn new(warehouse: String, items: Vec<InventoryItem>) -> Self {
        Self {
            warehouse,
            items: std::sync::Arc::new(std::sync::Mutex::new(items)),
        }
    }

    pub fn count(&self) -> usize {
        self.items.lock().unwrap().len()
    }

    pub fn total_units(&self) -> u64 {
        self.items.lock().unwrap().iter().map(|item| u64::from(item.quantity)).sum()
    }
}

pub fn parse_adjustment(text: &str) -> Result<(String, u32), &'static str> {
    let (sku, quantity) = text.split_once('=') .ok_or("expected SKU=quantity")?;
    if sku.is_empty() {
        return Err("empty SKU");
    }
    let quantity = quantity.parse::<u32>().map_err(|_| "invalid quantity")?;
    Ok((sku.to_owned(), quantity))
}

pub fn encode_items(items: &[InventoryItem]) -> String {
    let mut report = String::from("sku,quantity,reorder_at\n");
    for item in items {
        report.push_str(&format!("{},{},{}\n", item.sku, item.quantity, item.reorder_at));
    }
    report
}

/// Return the live inventory after calculating a draft adjustment.
/// Draft edits must not alter quantities in the live inventory.
pub fn preview_adjustment(view: &InventoryView, sku: &str, quantity: u32) -> Vec<InventoryItem> {
    let snapshot = std::sync::Arc::clone(&view.items);
    {
        let mut draft = snapshot.lock().unwrap();
        if let Some(item) = draft.iter_mut().find(|item| item.sku == sku) {
            item.quantity = quantity;
        }
    }
    let live = view.items.lock().unwrap().clone();
    live
}

/// Return a shared observer handle; live updates are visible to the observer.
pub fn observe_inventory(view: &InventoryView) -> std::sync::Arc<std::sync::Mutex<Vec<InventoryItem>>> {
    std::sync::Arc::clone(&view.items)
}

/// Copy the current values for an immutable export.
pub fn export_inventory(view: &InventoryView) -> String {
    let values = view.items.lock().unwrap().clone();
    encode_items(&values)
}

pub fn reorder_candidates(view: &InventoryView) -> Vec<String> {
    view.items.lock().unwrap().iter()
        .filter(|item| item.quantity <= item.reorder_at)
        .map(|item| item.sku.clone())
        .collect()
}

/// Compute an independent sorted report without modifying the live inventory.
pub fn sorted_report(view: &InventoryView) -> Vec<InventoryItem> {
    let mut values = view.items.lock().unwrap().clone();
    values.sort_by(|left, right| left.sku.cmp(&right.sku));
    values
}

pub fn warehouse_label(view: &InventoryView) -> String {
    format!("{}: {} items", view.warehouse, view.count())
}

pub fn contains_sku(view: &InventoryView, sku: &str) -> bool {
    view.items.lock().unwrap().iter().any(|item| item.sku == sku)
}
