def calculate_subtotal(quantity, unit_price); quantity * unit_price; end

def apply_tax(amount); amount + amount * 8 / 100; end

def build_total(quantity, unit_price)
  subtotal = calculate_subtotal(quantity, unit_price)
  apply_tax(subtotal)
end
