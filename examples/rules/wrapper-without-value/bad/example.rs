fn fetch_points(user_id: i32) -> i32 {
	user_id * 2
}

fn get_user_points(user_id: i32) -> i32 {
	fetch_points(user_id)
}
