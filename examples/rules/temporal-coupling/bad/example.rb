class Rectangle
  def initialize
    @initialized = false
  end

  def init(width, height)
    @width = width
    @height = height
    @initialized = true
  end

  def area
    return 0 unless @initialized

    @width * @height
  end
end
