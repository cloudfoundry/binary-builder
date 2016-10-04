# encoding: utf-8
class DetermineChecksum
  def initialize(options)
    @options = options
  end

  def to_h
    checksum_type = ([:md5, :sha256, :gpg, :git] & @options.keys).first
    {
      checksum_type => @options[checksum_type]
    }
  end
end
