# Homebrew formula template.
#
# Publish this file as Formula/jevlint.rb in a `homebrew-jevlint` tap repo.
# Before publishing, set `url` to a release tag tarball and replace `sha256`
# with the checksum of that tarball.
class Jevlint < Formula
  desc "Check code against plain-language rules with Jev"
  homepage "https://github.com/codegirl-007/jevlint"
  url "https://github.com/codegirl-007/jevlint/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "REPLACE_WITH_SHA256_OF_THE_TAGGED_TARBALL"
  license "MIT"
  head "https://github.com/codegirl-007/jevlint.git", branch: "master"

  depends_on "go" => :build

  def install
    ldflags = "-s -w -X github.com/codegirl-007/jevlint/internal/cli.version=#{version}"
    system "go", "build", *std_go_args(ldflags: ldflags), "./cmd/jevlint"
  end

  test do
    system bin/"jevlint", "version"
  end
end
