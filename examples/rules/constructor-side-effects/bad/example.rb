class Config
  def initialize(path)
    @handle = File.open(path)
  end
end
