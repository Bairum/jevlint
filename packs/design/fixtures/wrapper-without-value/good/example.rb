def fetch_points(user_id)
  user_id * 2
end

def get_user_points(user_id)
  return 0 if user_id <= 0
  fetch_points(user_id)
end
