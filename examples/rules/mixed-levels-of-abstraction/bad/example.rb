def apply_tax(amount); amount + amount * 8 / 100; end

def build_total(quantity, unit_price)
  subtotal = quantity * unit_price
  discount = subtotal / 10
  taxed = apply_tax(subtotal - discount)
  taxed + 250
end
