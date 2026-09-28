MAX_RETRY_ATTEMPTS = 3

def can_retry(attempts)
  attempts < MAX_RETRY_ATTEMPTS
end
