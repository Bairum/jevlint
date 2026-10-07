START = 1
STOP = 2

def run(command)
  case command
  when START then 1
  when STOP then 0
  else -1
  end
end
