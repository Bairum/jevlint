def count_positive_pairs(values)
  pairs = 0
  values.each_index do |i|
    if values[i] > 0
      values.drop(i + 1).each do |value|
        if value > 0
          pairs += 1
        end
      end
    end
  end
  pairs
end
