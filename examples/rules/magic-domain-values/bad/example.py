def can_retry(attempts):
    if attempts < 0:
        return False
    return attempts < 3
