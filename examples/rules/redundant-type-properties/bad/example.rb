class Rectangle
  attr_reader :width, :height, :area

  def initialize(width, height)
    @width = width
    @height = height
    @area = width * height
  end
end
