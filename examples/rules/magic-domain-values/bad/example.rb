def can_retry(attempts)
  return false if attempts < 0
  attempts < 3
end
