def shipping_cost(total):
    free_by_total = total >= 100.0
    free_by_subtotal = total > 99.99
    if free_by_total and free_by_subtotal:
        return 0.0
    return 5.0
