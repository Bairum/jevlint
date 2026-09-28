def persist(value)
  value * 2
end
class Store
  def save(value)
    persist(value)
  end
end
