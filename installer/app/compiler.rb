require 'bundler/setup'
require 'sprockets'
require 'es6-module-mapper'
require 'react-jsx-sprockets'
require 'marbles-js'
require 'fileutils'

module Installer
  def self.input_dir
    @input_dir ||= File.join(File.dirname(__FILE__), 'src')
  end

  def self.vendor_dir
    @vendor_dir ||= File.join(File.dirname(__FILE__), 'vendor')
  end

  def self.output_dir
    @output_dir ||= File.join(File.dirname(__FILE__), 'build')
  end

  def self.sprockets_environment
    if @sprockets_environment
      return @sprockets_environment
    end

    # Setup Sprockets Environment
    @sprockets_environment = ::Sprockets::Environment.new do |env|
      env.logger = Logger.new(STDOUT)
      env.context_class.class_eval do
        include MarblesJS::Sprockets::Helpers
      end

      # we're not using the directive processor, so unregister it
      env.unregister_preprocessor(
        'application/javascript', ::Sprockets::DirectiveProcessor)
    end
    MarblesJS::Sprockets.setup(@sprockets_environment)
    @sprockets_environment.append_path(input_dir)

    return @sprockets_environment
  end

  def self.compile
    manifest = ::Sprockets::Manifest.new(
      sprockets_environment,
      output_dir,
      File.join(output_dir, 'manifest.json')
    )

    manifest.compile(%w[application.js normalize.css application.css])

    manifest.assets.each_pair do |logical_path, digest_path|
      FileUtils.mv(File.join(output_dir, digest_path), File.join(output_dir, logical_path))
    end

    Dir[File.join(vendor_dir, "*")].each do |vendor_path|
      FileUtils.cp(vendor_path, File.join(output_dir, vendor_path.sub(vendor_dir, '')))
    end

    FileUtils.rm(manifest.filename)
  end
end
