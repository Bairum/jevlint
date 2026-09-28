def load
  begin
    connect
  rescue StandardError => error
    warn error.message
  end
end

def connect
end
