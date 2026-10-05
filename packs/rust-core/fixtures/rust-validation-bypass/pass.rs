/// A validated rate in 0..=100; every safe value represents a percentage.
#[derive(Clone, Copy)]
pub struct Percent {
    value: u8,
}

impl Percent {
    pub fn new(value: u8) -> Result<Self, &'static str> {
        if value > 100 {
            return Err("rate exceeds 100");
        }
        Ok(Self { value })
    }

    pub fn set(&mut self, value: u8) -> Result<(), &'static str> {
        let replacement = Self::new(value)?;
        *self = replacement;
        Ok(())
    }

    /// The remaining rate is in 0..=100.
    pub fn complement(&self) -> i16 {
        100 - i16::from(self.value)
    }

    pub fn value(&self) -> u8 {
        self.value
    }
}

#[derive(Clone)]
pub struct InvoiceLine {
    pub sku: String,
    pub quantity: u16,
    pub unit_cents: u32,
}

impl InvoiceLine {
    pub fn subtotal(&self) -> u64 {
        u64::from(self.quantity) * u64::from(self.unit_cents)
    }
}

pub struct Invoice {
    pub number: String,
    pub lines: Vec<InvoiceLine>,
    discount: Percent,
}

impl Invoice {
    pub fn new(number: String, lines: Vec<InvoiceLine>, discount: Percent) -> Self {
        Self { number, lines, discount }
    }

    pub fn subtotal(&self) -> u64 {
        self.lines.iter().map(InvoiceLine::subtotal).sum()
    }

    pub fn discount_rate(&self) -> u8 {
        self.discount.value()
    }

    pub fn line_count(&self) -> usize {
        self.lines.len()
    }
}

pub struct QuoteRecord {
    pub customer: String,
    pub discount: u8,
    pub lines: Vec<InvoiceLine>,
}

impl QuoteRecord {
    pub fn into_invoice(self, number: String) -> Result<Invoice, &'static str> {
        let discount = Percent::new(self.discount)?;
        Ok(Invoice::new(number, self.lines, discount))
    }
}

pub fn remaining_rate(raw: u8) -> Result<i16, &'static str> {
    let rate = Percent::new(raw)?;
    Ok(rate.complement())
}

pub fn invoice_label(invoice: &Invoice) -> String {
    format!("{}: {} lines", invoice.number, invoice.line_count())
}

pub fn sample_invoice() -> Result<Invoice, &'static str> {
    let lines = vec![
        InvoiceLine { sku: "FILTER".into(), quantity: 2, unit_cents: 850 },
        InvoiceLine { sku: "SEAL".into(), quantity: 6, unit_cents: 125 },
    ];
    let quote = QuoteRecord {
        customer: "Acme".into(),
        discount: 15,
        lines,
    };
    quote.into_invoice("INV-2026-104".into())
}

pub fn collect_skus(invoice: &Invoice) -> Vec<&str> {
    invoice.lines.iter().map(|line| line.sku.as_str()).collect()
}
