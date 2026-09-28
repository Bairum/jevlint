def retry_delay(attempt)
  # Delay grows with each attempt so clients do not retry together after an outage.
  100 * attempt * attempt
end
