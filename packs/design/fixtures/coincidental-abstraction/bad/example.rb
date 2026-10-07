def is_over_budget(amount)
  amount > 1000
end

def check_systems(balance, cpu_load)
  over_budget = is_over_budget(balance)
  overloaded = is_over_budget(cpu_load)
  over_budget || overloaded
end
