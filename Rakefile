# Build tasks for building the system
#

 # Ensure all our ruby code in the rake files (and friends) runs in a deterministic environment providede by Bundler.
require 'bundler/setup'

require_relative 'build/utils/disk_builder'
require_relative 'build/utils/image_builder'

namespace :build do

	ROCKETSHIP_COMPONENTS = [ 'commander', 'crashcorder', 'preflight', 'radio']

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
	desc 'Build the system image (params are bool,bool,string)'
	task :image, [:debug, :upgrade, :rootfs_tarball_path,] => copy_bin_tasks do |t, args|

		args.with_defaults(:debug               => false)
		args.with_defaults(:upgrade             => false)
		args.with_defaults(:rootfs_tarball_path => ImageBuilder::CACHED_ROOTFS_TGZ_PATH)

		# TODO: provide instructions on how to get a rootfs
		unless File.exists?(args.rootfs_tarball_path)
			raise ArgumentError, "No usable rootfs at #{args.rootfs_tarball_path}. Aborting"
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
	task :disk, [:image_file] do |t, args|
		args.with_defaults(:image_path  => ImageBuilder::ROCKETSHIP_IMAGE_FILE_PATH)
		args.with_defaults(:debug       => false)

		DiskBuilder.new(args.image_path, (args.debug and args.debug == 'true')).build
	end


end

# Clean tasks
namespace :clean do

	clean_bin_tasks = []

	# [INTERNAL] Tasks for cleaning built binaries (in the src dir).
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

	# [INTERNAL] Tasks for cleaning up copied binaries (in the rootfs dir).
	# (Intentionally missing descriptions so that they are omitted in the -T output. Instead see
	# the :allbins target that uses these).
	ROCKETSHIP_COMPONENTS.each do |component|
		taskname = "copied_#{component}"
		clean_bin_tasks << taskname
		task taskname do
			srcfile = File.join(File.dirname(__FILE__), 'bin', component, component)
			sh("rm -f #{srcfile}")
		end
	end

	# User facing task that cleans up all the binaries
	desc "Clean all built binaries"
	task :allbins => clean_bin_tasks

	desc 'Clean everything'
	task :full => :allbins do
		# bins (built, copied) will be cleaned by dependent task.
		# Clean up image files, disk files
		sh("rm -f build/rocketship.img")
		sh("rm -f build/rocketship.vmdk")
	end
end
