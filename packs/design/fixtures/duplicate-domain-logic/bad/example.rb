def shipping_cost(total)
  free_by_total = total >= 100.0
  free_by_subtotal = total > 99.99
  if free_by_total && free_by_subtotal
    0.0
  else
    5.0
  end
end
