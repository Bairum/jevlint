def clamp(value, low, high)
  return low if value < low
  return high if value > high
  value
end
