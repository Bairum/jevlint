def fetch_points(user_id):
    return user_id * 2


def get_user_points(user_id):
    if user_id <= 0:
        return 0
    return fetch_points(user_id)
