def run(command)
  case command
  when "start" then 1
  when "stop" then 0
  else -1
  end
end
