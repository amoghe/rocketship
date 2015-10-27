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

	# These packages go into the barebones linux rootfs
	ESSENTIAL_PKGS = [
		'dbus'           ,
		'iputils-ping'   , # ping
		'isc-dhcp-client', # dhcp
		'logrotate'      ,
		'net-tools'      , # ifconfig
		'rsyslog'        ,
		'openssh-server' ,
		'wget'           ,
	]

	# These are packages that are installed when transforming a basic rootfs into a rocketship rootfs.
	ADDITIONAL_PACKAGES = [
		'ca-certificates',
		'collectd-core'  ,
	]

	# These are packages installed when we detect a developer build.
	DEV_BUILD_PKGS = [
		'emacs24-nox',
		'sudo',
		'lsof',
	]

	COMMON_APT_OPTS = [
		'--yes',
		'--no-install-recommends',
	].join(' ')


	UBUNTU_APT_ARCHIVE_URL = "http://archive.ubuntu.com/ubuntu"

	CWD = File.dirname(__FILE__)
	BUILD_DIR_PATH = File.expand_path(File.join(CWD, '..'))
	CACHE_DIR_PATH = File.expand_path(File.join(BUILD_DIR_PATH, "cache"))
	ROCKETSHIP_ROOTFS_DIR_PATH = File.join(BUILD_DIR_PATH, 'rootfs')

	CACHED_DEBOOTSTRAP_PKGS_NAME = "debootstrap_pkgs.tgz"
	CACHED_DEBOOTSTRAP_PKGS_PATH = File.join(CACHE_DIR_PATH, CACHED_DEBOOTSTRAP_PKGS_NAME)

	UBUNTU_ROOTFS_TGZ_NAME = "ubuntu_#{UBUNTU_RELEASE}.tar.gz"
	UBUNTU_ROOTFS_TGZ_PATH = File.join(CACHE_DIR_PATH, UBUNTU_ROOTFS_TGZ_NAME)

	ROCKETSHIP_IMAGE_FILE_NAME = 'rocketship.img'
	ROCKETSHIP_IMAGE_FILE_PATH = File.join(BUILD_DIR_PATH, ROCKETSHIP_IMAGE_FILE_NAME)

	attr_reader :rootfs
	attr_reader :dev_build
	attr_reader :upgrade

	def initialize(rootfs_tgz_path, opts={})
		@rootfs    = (rootfs_tgz_path && File.exists?(rootfs_tgz_path)) ? rootfs_tgz_path : nil
		@dev_build = !!opts[:dev_build]
		@upgrade   = !!opts[:upgrade]
	end

	##
	# Build the image
	#
	def build
		self.ensure_root_privilege

		banner('Build options')
		if rootfs == nil
			info("Build debootstrap rootfs\t: true")
		else
			info("Using rootfs tarball at\t: #{File.expand_path(@rootfs)}")
		end

		info("Install additional packages\t: #{dev_build}")
		info("Upgrade the distribution\t: #{upgrade}")
		sleep(1) # Let it sink in

		self.on_mounted_tmpfs do |tempdir|

			begin
				if rootfs == nil
					banner("Creating debootstrap rootfs")
					create_debootstrap_rootfs(tempdir)
				else
					banner("Unpacking specified rootfs")
					self.extract_rootfs(tempdir)
				end

				banner('Updating rootfs')
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

	def create_debootstrap_rootfs(tempdir)
		if File.exists?(CACHED_DEBOOTSTRAP_PKGS_PATH)
			cached_pkgs_opt = "--unpack-tarball=#{CACHED_DEBOOTSTRAP_PKGS_PATH}"
			info("Cached debootstrap packages found in tarball at: #{CACHED_DEBOOTSTRAP_PKGS_PATH}")
		else
			cached_pkgs_opt = ""
			info("No cached debootstrap packages found.")
		end

		execute!(["debootstrap",
			  "--variant minbase",
			  cached_pkgs_opt,
			  "--include #{ESSENTIAL_PKGS.join(",")}",
			  "trusty",
			  tempdir,
			  UBUNTU_APT_ARCHIVE_URL,
		].join(" "))
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

		trusty_update_repo = "deb http://us.archive.ubuntu.com/ubuntu/ trusty-updates main restricted"
		trusty_universe_repo = "deb http://us.archive.ubuntu.com/ubuntu/ trusty universe"

		chroot_cmds = [
			"mkdir -p #{DiskBuilder::CONFIG_PARTITION_MOUNT}",

			# put the version into the image (ctime is build time)
			"echo #{IMAGE_VERSION} > /etc/rocketship_version",

			# add nameservers so that apt will work
			'echo nameserver 8.8.8.8 > /etc/resolv.conf',

			# ensure no services are started in the chroot
			'echo -e \'#!/bin/bash\nexit 101\' > /usr/sbin/policy-rc.d',
			'chmod a+x /usr/sbin/policy-rc.d',

			# Add more repos
			"echo #{trusty_update_repo} >> /etc/apt/sources.list",
			"echo #{trusty_universe_repo} >> /etc/apt/sources.list",

			# Update the apt cache
			'apt-get update',

			# Get latest upgrades
			upgrade ? 'apt-get --yes upgrade' : '',

			# mount proc, kernel installation needs it
			'mount -t proc none /proc',

			# Kernel
			"apt-get #{COMMON_APT_OPTS} install linux-image-generic",

			# unmount proc
			'umount /proc',

			# Download essential packages
			"apt-get #{COMMON_APT_OPTS} install #{ESSENTIAL_PKGS.join(' ')}",

			# Clean up the apt cache, reduces the img size
			'apt-get clean',

			# Undo the hacks - in reverse order
			'rm -f /usr/sbin/policy-rc.d',
		].reject(&:empty?)

		chroot_cmds.each_with_index do |cmd, num|
			# If its an apt command, run it non interactively
			cmd = "DEBIAN_FRONTEND=noninteractive #{cmd}" if cmd.include?('apt-get')
			#info("[Step #{num+1} of #{chroot_cmds.length}] #{cmd}")
			execute!("chroot #{rootfs_dir} /bin/bash -c \"#{cmd}\"", true)
		end

		nil
	end

	##
	# Customize the rootfs with additional packages and files.
	#
	def customize(rootfs_dir)
		info("Installing additional packages")
		chroot_cmds = [
			# Additional packages
			"apt-get #{COMMON_APT_OPTS} install #{ADDITIONAL_PACKAGES.join(' ')}",

			# Download additional developer packages
			dev_build ? "apt-get #{COMMON_APT_OPTS} install #{DEV_BUILD_PKGS.join(' ')}" : '',

			# Download and install influxdb
			"wget https://s3.amazonaws.com/influxdb/influxdb_0.9.4.2_amd64.deb -O /tmp/influxdb.deb",
			"dpkg -i /tmp/influxdb.deb",
			"rm -f /tmp/influxdb.deb",
		].reject(&:empty?)

		chroot_cmds.each_with_index do |cmd, idx|
			# If its an apt command, run it non interactively
			cmd = "DEBIAN_FRONTEND=noninteractive #{cmd}" if cmd.include?('apt-get')
			#info("[Step #{num+1} of #{chroot_cmds.length}] #{cmd}")
			execute!("chroot #{rootfs_dir} /bin/bash -c \"#{cmd}\"", true)
		end

		info('Moving parts from warehouse into target')
		# We need to perform the copy as root, since the dest dir is owned by root.
		execute!("cp -r #{File.join(ROCKETSHIP_ROOTFS_DIR_PATH, '.')} #{rootfs_dir}")
	end

	##
	# Pack up the system image (rootfs) into a single file we can ship
	#
	def package(rootfs_dir)

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

	##
	# Create a debootstrap compatible tarball of deb packages.
	#
	def create_debootstrap_packages_tarball()
		cached_pkgs_tarball = CACHED_DEBOOTSTRAP_PKGS_PATH

		banner("Removing old cached packages")
		execute!("rm -f #{cached_pkgs_tarball}")

		self.on_mounted_tmpfs do |tempdir|
                        # create a work dir in the tempdir, because debootstrap wants to delete its work dir when
                        # it finishes, but the tempdir is owned by root.

                        workdir = File.join(tempdir, "work")
                        execute!("mkdir -p #{workdir}")

			banner("Invoking debootstrap to create new cached packages tarball")
			execute!(["debootstrap",
				"--variant minbase",
				"--include #{ESSENTIAL_PKGS.join(",")}",
				"--make-tarball #{cached_pkgs_tarball}",
				"trusty",
				workdir,
				UBUNTU_APT_ARCHIVE_URL,
			].join(" "))
		end

		banner("debootstrap packages cached at:" + cached_pkgs_tarball)
	end
end
