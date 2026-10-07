def is_over_budget(amount):
    return amount > 1000


def check_systems(balance, cpu_load):
    over_budget = is_over_budget(balance)
    overloaded = is_over_budget(cpu_load)
    return over_budget or overloaded
