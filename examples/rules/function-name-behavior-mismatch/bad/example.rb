class User
  def initialize(name)
    @name = name
    @reads = 0
  end

  def get_name
    @reads += 1
    @name
  end
end
