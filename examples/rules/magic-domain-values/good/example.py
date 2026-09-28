MAX_RETRY_ATTEMPTS = 3


def can_retry(attempts):
    return attempts < MAX_RETRY_ATTEMPTS
