def calculate_subtotal(quantity, unit_price):
    return quantity * unit_price

def apply_tax(amount):
    return amount + amount * 8 // 100

def build_total(quantity, unit_price):
    subtotal = calculate_subtotal(quantity, unit_price)
    return apply_tax(subtotal)
