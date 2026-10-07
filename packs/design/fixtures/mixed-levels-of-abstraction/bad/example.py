def apply_tax(amount):
    return amount + amount * 8 // 100

def build_total(quantity, unit_price):
    subtotal = quantity * unit_price
    discount = subtotal // 10
    taxed = apply_tax(subtotal - discount)
    return taxed + 250
