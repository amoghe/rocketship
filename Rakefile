# Rake tasks for building the various parts of the system
#

# Ensure all our ruby code in the rake files (and friends) run in
# a deterministic environment providede by Bundler.
require 'bundler/setup'

require_relative 'build/utils/disk_builder'
require_relative 'build/utils/image_builder'
require_relative 'build/utils/debootstrap_builder'

namespace :build do

	def binpath(name)                 ; File.join("bin", name, name)                      ; end
	def copied_binpath(name)          ; File.join('build/rootfs/bin', name)               ; end
	def shellcmd_binpath(name)        ; File.join("bin/shellcommands", name, name)        ; end
	def copied_shellcmd_binpath(name) ; File.join('build/rootfs/opt/shellcommands', name) ; end

	# Names of component binaries
	ROCKETSHIP_COMPONENTS = [
		'commander' ,
		'crashcorder',
		'preflight'  ,
		'radio'      ,
		'shell'      ,
	]

	# Names of shell commands
	ROCKETSHIP_SHELLCMDS = [
		'hostname'   ,
		'interfaces' ,
		'users'      ,
	]

	ALL_ROCKETSHIP_BINPATHS        = ROCKETSHIP_COMPONENTS.map { |comp| binpath(comp) }
	ALL_ROCKETSHIP_BINPATHS_COPIED = ROCKETSHIP_COMPONENTS.map { |comp| copied_binpath(comp) }

	ALL_SHELLCMD_BINPATHS          = ROCKETSHIP_SHELLCMDS.map { |cmd| shellcmd_binpath(cmd) }
	ALL_SHELLCMD_BINPATHS_COPIED   = ROCKETSHIP_SHELLCMDS.map { |cmd| copied_shellcmd_binpath(cmd) }

	#
	# Tasks for building various files (intentionally not given descriptions, so that they are suppressed
	# in the -T output). Instead, see the :allbins target which builds these.
	#

	ROCKETSHIP_COMPONENTS.each do |component|
		# How to build the component binaries.
		file binpath(component) do
			subdir = File.dirname(binpath(component))
			sh("cd #{subdir}; go get && go build")
		end

		# How to copy it into the rootfs
		file copied_binpath(component) => binpath(component) do
			sh("mkdir -p #{File.dirname(copied_binpath(component))}")
			sh("cp #{binpath(component)} #{copied_binpath(component)}")
		end
	end

	ROCKETSHIP_SHELLCMDS.each do |cmd_name|
		# How to build the shell command binaries.
		file shellcmd_binpath(cmd_name) do
			subdir = File.dirname(shellcmd_binpath(cmd_name))
			sh("cd #{subdir} && go get && go build")
		end

		# How to copy it into the rootfs
		file copied_shellcmd_binpath(cmd_name) => shellcmd_binpath(cmd_name) do
			sh("mkdir -p #{File.dirname(copied_shellcmd_binpath(cmd_name))}")
			sh("cp #{shellcmd_binpath(cmd_name)} #{copied_shellcmd_binpath(cmd_name)}")
		end
	end


	# How to build up a cache of packages needed for speeding up repeated debootstrap runs.
	file DebootstrapBuilder::CACHED_DEBOOTSTRAP_PKGS_PATH do
		DebootstrapBuilder.new(ENV.has_key?('VERBOSE')).create_debootstrap_packages_tarball()
	end

	# How to build a basic rootfs using debootstrap.
	# This relies on a tarball of cached packages that is usable by debootstrap.
	file DebootstrapBuilder::DEBOOTSTRAP_ROOTFS_PATH => DebootstrapBuilder::CACHED_DEBOOTSTRAP_PKGS_PATH do
		DebootstrapBuilder.new(ENV.has_key?('VERBOSE')).create_debootstrap_rootfs()
	end

	# How to build a rocketship rootfs/image using a debootstrap rootfs.
	image_deps = \
		ALL_ROCKETSHIP_BINPATHS_COPIED + \
		ALL_SHELLCMD_BINPATHS_COPIED + \
		[DebootstrapBuilder::DEBOOTSTRAP_ROOTFS_PATH]
	file ImageBuilder::ROCKETSHIP_IMAGE_FILE_PATH => image_deps do
		dev_build = ENV.has_key?('DEV')
		verbose   = ENV.has_key?('VERBOSE')
		ImageBuilder.new(DebootstrapBuilder::DEBOOTSTRAP_ROOTFS_PATH, dev_build, verbose).build()
	end

	# How to build a disk (vmdk) given a rocketship rootfs/image.
	file DiskBuilder::VMDK_FILE_PATH => ImageBuilder::ROCKETSHIP_IMAGE_FILE_PATH do
		DiskBuilder.new(ImageBuilder::ROCKETSHIP_IMAGE_FILE_PATH, ENV.has_key?('VERBOSE')).build
	end

	#
	# Build ALL binaries
	#
	desc "Build all binaries"
	task :allbins => ALL_ROCKETSHIP_BINPATHS + ALL_SHELLCMD_BINPATHS

	#
	# Copy all binaries into the rootfs.
	#
	desc "Copy binaries into rootfs in preparation for image build"
	task :copybins => ALL_ROCKETSHIP_BINPATHS_COPIED + ALL_SHELLCMD_BINPATHS_COPIED

	#
	# Build a tarball of cached deb packages usable by debootstrap (created by debootstrap).
	#
	desc 'Build debootstrap cache (env vars: VERBOSE)'
	task :debootstrap_cache => DebootstrapBuilder::CACHED_DEBOOTSTRAP_PKGS_PATH

	#
	# Build a basic rootfs using debootstrap.
	#
	desc 'Build basic rootfs using debootstrap (env vars: VERBOSE)'
	task :debootstrap_rootfs => DebootstrapBuilder::DEBOOTSTRAP_ROOTFS_PATH

	#
	# Build image.
	#
	desc 'Build the rocketship image (env vars: DEV, VERBOSE)'
	task :image => ImageBuilder::ROCKETSHIP_IMAGE_FILE_PATH

	#
	# Build disk.
	#
	desc 'Build a bootable disk containing the rocketship image (env vars: VERBOSE)'
	task :disk => DiskBuilder::VMDK_FILE_PATH
end

# Clean tasks
namespace :clean do

	desc "Clean all built binaries"
	task :allbins do
		ROCKETSHIP_COMPONENTS.map{|comp| binpath(comp)}.each{|filepath| sh("rm -f #{filepath}")}
		ROCKETSHIP_SHELLCMDS.map{|cmd| shellcmd_binpath(cmd)}.each{|filepath| sh("rm -f #{filepath}")}
	end

	desc "Clean binaries copied into rootfs during image builds"
	task :copiedbins do
		ROCKETSHIP_COMPONENTS.map{|comp| copied_binpath(comp)}.each{|filepath| sh("rm -f #{filepath}")}
		ROCKETSHIP_SHELLCMDS.map{|cmd| copied_shellcmd_binpath(cmd)}.each{|filepath| sh("rm -f #{filepath}")}
	end

	desc "Clean the debootstrap rootfs file"
	task :debootstrap_rootfs do
		sh("rm -f #{DebootstrapBuilder::DEBOOTSTRAP_ROOTFS_PATH}")
	end

	desc "Clean the rocketship image file"
	task :image do
		sh("rm -f #{ImageBuilder::ROCKETSHIP_IMAGE_FILE_PATH}")
	end

	desc "Clean the disk file"
	task :disk do
		sh("rm -f #{DiskBuilder::VMDK_FILE_PATH}")
	end

	desc 'Clean everything'
	task :full => [:allbins, :copiedbins, :debootstrap_rootfs, :image, :disk]
end

namespace :test do

	desc 'all go code'
	task :go => "build:allbins" do
		ROCKETSHIP_COMPONENTS.reject { |c| c == "preflight" }.each do |comp|
			sh("cd #{comp} && go test ./... -check.v")
		end
	end
end
