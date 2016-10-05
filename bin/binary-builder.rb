#!/usr/bin/env ruby
# encoding: utf-8

require 'bundler'
require 'optparse'
require_relative '../lib/yaml_presenter'
require_relative '../lib/archive_recipe'
Dir['recipe/*.rb'].each { |f| require File.expand_path(f) }

recipes = {
     'ruby' => RubyRecipe,
     'bundler' => BundlerRecipe,
     'node' => NodeRecipe,
     'jruby' => JRubyMeal,
     'httpd' => HTTPdMeal,
     'python' => PythonRecipe,
     'php' => Php5Meal,
     'php7' => Php7Meal,
     'nginx' => NginxRecipe,
     'godep' => GodepMeal,
     'glide' => GlideRecipe,
     'go' => GoRecipe,
     'dotnet' => DotNetRecipe
}

options = {}
optparser = OptionParser.new do |opts|
  opts.banner = 'USAGE: binary-builder [options] (A checksum method is required)'

  opts.on('-nNAME', '--name=NAME', "Name of the binary.  Options: [#{recipes.keys.join(", ")}]") do |n|
    options[:name] = n
  end
  opts.on('-vVERSION', '--version=VERSION', 'Version of the binary e.g. 1.7.11') do |n|
    options[:version] = n
  end
  opts.on('--sha256=SHA256', 'SHA256 of the binary ') do |n|
    options[:sha256] = n
  end
  opts.on('--md5=MD5', 'MD5 of the binary ') do |n|
    options[:md5] = n
  end
  opts.on('--gpg-rsa-key-id=RSA_KEY_ID', 'RSA Key Id e.g. 10FDE075') do |n|
    options[:gpg] ||= {}
    options[:gpg][:key] = n
  end
  opts.on('--gpg-signature=ASC_KEY', 'content of the .asc file') do |n|
    options[:gpg] ||= {}
    options[:gpg][:signature] = n
  end
  opts.on('--git-commit-sha=SHA', 'git commit sha of the specified version') do |n|
    options[:git] ||= {}
    options[:git][:commit_sha] = n
  end
end
optparser.parse!

unless options[:name] && options[:version] && (
    options[:sha256] ||
    options[:md5] ||
    (options.has_key?(:git) && options[:git][:commit_sha]) ||
    (options[:gpg][:signature] && options[:gpg][:key])
)
  raise optparser.help
end

raise "Unsupported recipe [#{options[:name]}], supported options are [#{recipes.keys.join(", ")}]" unless recipes.has_key?(options[:name])

recipe = recipes[options[:name]].new(
  options[:name],
  options[:version],
  DetermineChecksum.new(options).to_h
)
Bundler.with_clean_env do
  puts "Source URL: #{recipe.url}"

  recipe.cook
  ArchiveRecipe.new(recipe).tar!

  puts 'Source YAML:'
  puts YAMLPresenter.new(recipe).to_yaml
end
