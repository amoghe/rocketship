require 'open3'
require 'pp'

require_relative 'base_builder'
require_relative 'disk_builder' # So we access its constants

#- - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
# Module that contains the routines used to build images.
#
class ImageBuilder < BaseBuilder

	UBUNTU_RELEASE	= '14_04_03'
	IMAGE_VERSION 	= '0.3.0'

	ESSENTIAL_ADDON_PKGS = [
		'dbus'           ,
		'iputils-ping'   , # ping
		'isc-dhcp-client', # dhcp
		#'libsqlite3-dev' , # activerecord
		'logrotate'      ,
		'net-tools'      , # ifconfig
		'rsyslog'        ,
		#'ruby1.9.1'      , # ruby 1.9.3
		'openssh-server' ,
	]

	DEBUGGING_PKGS = [
		'emacs24-nox',
		'sudo',
		'lsof',
	]

	CWD = File.dirname(__FILE__)
	BUILD_DIR_PATH = File.expand_path(File.join(CWD, '..'))

	CACHED_ROOTFS_TGZ_NAME = "ubuntu_#{UBUNTU_RELEASE}.tar.gz"
	CACHED_ROOTFS_TGZ_PATH = File.join(BUILD_DIR_PATH, 'cache', CACHED_ROOTFS_TGZ_NAME)

	ROCKETSHIP_ROOTFS_DIR_PATH = File.join(BUILD_DIR_PATH, 'rootfs')

	ROCKETSHIP_IMAGE_FILE_NAME = 'rocketship.img'
	ROCKETSHIP_IMAGE_FILE_PATH = File.join(BUILD_DIR_PATH, ROCKETSHIP_IMAGE_FILE_NAME)

	attr_reader :rootfs
	attr_reader :debug
	attr_reader :upgrade

	def initialize(rootfs_tarball_path, opts={})
		raise ArgumentError, 'ENOENT' unless File.exists?(rootfs_tarball_path)
		@rootfs = rootfs_tarball_path

		@debug   = opts[:debug]
		@upgrade = opts[:upgrade]
	end

	##
	# Build the image
	#
	def build
		self.ensure_root_privilege

		banner('Build options')
		info("Install additional packages	: #{debug}")
		info("Dist upgrade the rootfs		: #{upgrade}")
		sleep(1) # Time to register

		self.on_mounted_tmpfs do |tempdir|

			begin
				banner("Unpacking rootfs (from #{rootfs})")
				self.extract_rootfs(tempdir)

				banner('Update rootfs')
				self.install_additional_packages(tempdir)

				banner('Customize image (with rocketship components)')
				self.customize(tempdir)

				banner('Packaging the image')
				self.package(tempdir)
			rescue => e
				warn(e)
				banner('Failed')
				pp e.backtrace unless (e.is_a?(ArgumentError) or e.is_a?(PermissionError))
			else
				banner('Done')
			end

		end # on_mounted_tmpfs

		nil
	end

	##
	# Extract the rootfs tarball into the specified (temp)dir.
	#
	def extract_rootfs(tempdir)

		# TODO: make this a constant
		exclude_dirs = ['etc/init']

		# Unpack the rootfs
		execute!([ 'tar'                ,
			'--extract'          , # -x
			'--gunzip'           , # -z
			'--numeric-owner'    , # dont lookup /etc/passwd
			'--preserve-permissions',
			"--file=#{rootfs}"   , # -f
			exclude_dirs.map { |dir| "--exclude=#{dir}" } ,
			"-C \"#{tempdir}\""  , # untar into tmpdir
			]         \
			.flatten  \
			.join(' '),

			true) # Needs sudo due to tarball containing special device files
			nil
	end

	##
	# Update the distro and install additional packages (in the rootfs)
	#
	def install_additional_packages(rootfs_dir)

		apt_opts = [
			'--yes',
			'--no-install-recommends',
		].join(' ')

		chroot_cmds = [

			"mkdir -p #{DiskBuilder::CONFIG_PARTITION_MOUNT}",

			# put the version into the image (ctime is build time)
			"echo #{IMAGE_VERSION} > /etc/rocketship_version",

			# add nameservers so that apt will work
			'echo nameserver 8.8.8.8 > /etc/resolv.conf',

			# ensure no services are started in the chroot
			'dpkg-divert --local --rename --add /sbin/initctl',
			'ln -s /bin/true /sbin/initctl',

			# Update the apt cache
			'apt-get update',

			# Get latest upgrades
			upgrade ? 'apt-get --yes upgrade' : '',

			# mount proc, kernel installation needs it
			'mount -t proc none /proc',

			# Kernel
			"apt-get #{apt_opts} install linux-image-generic",

			# unmount proc
			'umount /proc',

			# Download essential packages
			"apt-get #{apt_opts} install #{ESSENTIAL_ADDON_PKGS.join(' ')}",

			# Download additional debugging packages
			debug ? "apt-get #{apt_opts} install #{DEBUGGING_PKGS.join(' ')}" : '',

			# Clean up the apt cache, reduces the img size
			'apt-get clean',

			# Undo the hack
			'rm /sbin/initctl',
			'dpkg-divert --local --rename --remove /sbin/initctl',
		].reject(&:empty?)

		Open3.popen3("sudo chroot #{rootfs_dir}") do |sin, sout, serr, stat|

			# Print stuff for liveness
			Thread.new { sout.each { |line| puts line } }

			# Since we're in interactive mode this is the easiest way to detect
			# individual command failures and bail on the first error. Else we
			# run the risk of proceeding to create a faulty image.
			sin.puts('set -e')
			# make the subshell print it so it shows in our output (which is being
			# provided by the thread draining its stdout).
			sin.puts('set -x')

			chroot_cmds.each_with_index do |cmd, num|
				# If its an apt command, run it non interactively
				cmd = "DEBIAN_FRONTEND=noninteractive #{cmd}" if cmd.include?('apt-get')
				#info("[Step #{num+1} of #{chroot_cmds.length}] #{cmd}")
				sin.puts(cmd)
			end

			sin.puts('exit')

			res = stat.value

			if not res.success?
				warn('ERROR while installing additional packages in rootfs, stderr output:')
				warn(serr.read)
				raise RuntimeError, 'rootfs customization failed'
			end

		end

		nil
	end

	##
	# Customize the rootfs with our files.
	#
	def customize(rootfs_dir)
		info('Moving parts from warehouse into target')
		# We need to perform the copy as root, since the dest dir is owned by root.
		execute!("cp -r #{File.join(ROCKETSHIP_ROOTFS_DIR_PATH, '.')} #{rootfs_dir}")
	end

	##
	# Pack up the system image (rootfs) into a single file we can ship
	#
	def package(rootfs_dir)
		# info('Setting up permissions on target')
		# orig_uid = ENV['SUDO_UID']
		# orig_gid = ENV['SUDO_GID']

		cmd = [ 'tar ',
			'--create',
			'--gzip',
			"--file=#{ROCKETSHIP_IMAGE_FILE_PATH}",
			# TODO: preserve perms, else whoever uses the image will have to twidle the perms again.
			#'--owner=0',
			#'--group=0',
			'--preserve-permissions',
			'--numeric-owner',
			"-C #{rootfs_dir} ."
		].join(' ')

		info('Packaging...')
		execute!(cmd, true)

		nil
	end

end
