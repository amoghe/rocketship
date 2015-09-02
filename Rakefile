# Build tasks for building the system
#

 # Ensure all our ruby code in the rake files (and friends) runs in a deterministic environment providede by Bundler.
require 'bundler/setup'

require_relative 'build/utils/disk_builder'
require_relative 'build/utils/image_builder'

namespace :build do

	ROCKETSHIP_COMPONENTS = [ 'commander', 'crashcorder', 'radio']

	build_bin_tasks = []
	copy_bin_tasks = []

	#
	# Tasks for building binaries (intentionally not given descriptions, so that they are suppressed
	# in the -T output). Instead, see the :allbins target which builds these.
	#
	ROCKETSHIP_COMPONENTS.each_with_index do |component, idx|
		taskname = component
		build_bin_tasks << taskname
		task taskname do |t|
			subdir = File.join(File.dirname(__FILE__), 'bin', component)

			# invoke the build in the subdir
			sh("$(cd #{subdir}; go build)")
		end
	end

	#
	# Tasks for copying component binaries (they depend on respective task to build binaries)
	# (intentionally not given descriptions so that they are suppressed in the -T output)
	#
	ROCKETSHIP_COMPONENTS.each_with_index do |component, idx|
		taskname = "copy_#{component}"
		copy_bin_tasks << taskname
		task taskname => component do |t|
			srcfile = File.join(File.dirname(__FILE__), 'bin', component, component)
			dstfile = File.join(File.dirname(__FILE__), 'build', 'rootfs', 'bin', component)

			sh("mkdir -p #{File.dirname(dstfile)}")
			sh("cp #{srcfile} #{dstfile}")
		end
	end

	#
	# Build ALL binaries (via dependencies)
	#
	desc "Build all binaries"
	task :allbins => build_bin_tasks

	#
	# Build image
	# (depends on the task that copies the binaries)
	#
	desc 'Build the system image (params are string,bool,bool)'
	task :image, [:debug, :upgrade, :rootfs_tarball_path,] do |t, args|

		args.with_defaults(:debug               => false)
		args.with_defaults(:upgrade             => false)
		args.with_defaults(:rootfs_tarball_path => ImageBuilder::CACHED_ROOTFS_TGZ_PATH)

		# TODO: provide instructions on how to get rootfs
		unless File.exists?(args.rootfs_tarball_path)
			raise ArgumentError, "- - -[ FATAL !!! ]- - - - - - - - -\n"\
			"No usable rootfs at #{args.rootfs_tarball_path}.\n"\
			"Please ensure either the default cached rootfs exists,\n"\
			"or specify the path to a custom rootfs file."
		end

		ImageBuilder.new(args.rootfs_tarball_path,
					:debug   => (args.debug   and args.debug   == 'true'),
					:upgrade => (args.upgrade and args.upgrade == 'true'),
		).build
	end

	#
	# Build disk (put image on specified disk device)
	# (depends on the task that builds the image).
	#
	desc 'Build a bootable disk containing the system image'
	task :disk, [:device_path, :image_file] do |t, args|

		args.with_defaults(:device_path => '/dev/sdb',
		:image_path  => ImageBuilder::ROCKETSHIP_IMAGE_FILE_PATH)

		args.with_defaults(:debug => false)

		DiskBuilder.new(args.device_path, args.image_path,
		:debug => (args.debug and args.debug == 'true')).build
	end


end

# Clean tasks
namespace :clean do

	clean_bin_tasks = []

	# Tasks for copying component binaries
	# (intentionally missing descriptions so that they are omitted in the -T output. Instead see
	# the :allbins target that uses these).
	ROCKETSHIP_COMPONENTS.each do |component|
		taskname = component
		clean_bin_tasks << taskname
		task taskname do |t|
			srcfile = File.join(File.dirname(__FILE__), 'bin', component, component)
			sh("rm -f #{srcfile}")
		end
	end

	desc "Clean all built binaries"
	task :allbins => clean_bin_tasks
end
