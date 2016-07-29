require 'open3'
require 'pp'

require_relative 'base_builder'
require_relative 'disk_builder' # So we can access its constants

#- - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
# Module that contains the routines used to build images.
#
class ImageBuilder < BaseBuilder

	IMAGE_VERSION 	= "0.3.0"

	# These are packages that are installed when transforming a basic rootfs into a rocketship rootfs.
	ADDITIONAL_PACKAGES = [
		"ca-certificates",
	]

	# These are packages installed when we detect a developer build.
	DEV_BUILD_PKGS = [
		"emacs24-nox",
		"sudo",
		"lsof",
	]

	CWD = File.dirname(__FILE__)
	BUILD_DIR_PATH = File.expand_path(File.join(CWD, ".."))
	CACHE_DIR_PATH = File.expand_path(File.join(BUILD_DIR_PATH, "cache"))
	ROCKETSHIP_ROOTFS_DIR_PATH = File.join(BUILD_DIR_PATH, "rootfs")

	ROCKETSHIP_IMAGE_FILE_NAME = "rocketship.img"
	ROCKETSHIP_IMAGE_FILE_PATH = File.join(BUILD_DIR_PATH, ROCKETSHIP_IMAGE_FILE_NAME)

	PROMETHEUS_URL = "https://github.com/prometheus/prometheus/releases/download/0.16.1/prometheus-0.16.1.linux-amd64.tar.gz"
	PROMETHEUS_EXPORTER_URL = "https://github.com/prometheus/node_exporter/releases/download/0.12.0rc1/node_exporter-0.12.0rc1.linux-amd64.tar.gz"

	attr_reader :verbose
	attr_reader :rootfs
	attr_reader :dev_build
	attr_reader :upgrade

	def initialize(rootfs_tgz_path, dev_build, verbose)
		raise ArgumentError, "Missing rootfs file: #{rootfs_tgz_path}" unless File.exists?(rootfs_tgz_path)

		@rootfs    = rootfs_tgz_path
		@dev_build = dev_build
		@upgrade   = false # TODO
		@verbose   = verbose
	end

	##
	# Build the image
	#
	def build
		header("Building system image")
		self.ensure_root_privilege

		notice('Build with these options')
		info("Install developer packages\t: #{dev_build}")
		info("Upgrade the distribution\t: #{upgrade}")
		sleep(1) # Let it sink in

		self.on_mounted_tmpfs do |tempdir|
			notice("Unpacking (debootstrap) rootfs")
			self.extract_rootfs(tempdir)

			notice('Installing additional components')
			self.install_additional_packages(tempdir)

			notice('Packaging the image')
			self.package(tempdir)

			notice('Done')
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

		common_apt_opts = [
			'--yes',
			'--no-install-recommends',
		].join(' ')

		mirror_url = "http://us.archive.ubuntu.com/ubuntu/"

		chroot_cmds = [
			"mkdir -p #{DiskBuilder::CONFIG_PARTITION_MOUNT}",

			# put the version into the image (ctime is build time)
			"echo #{IMAGE_VERSION} > /etc/rocketship_version",

			# add nameservers so that apt will work
			'echo nameserver 8.8.8.8 > /etc/resolv.conf',

			# ensure no services are started in the chroot
			'echo -e \'#!/bin/bash\nexit 101\' > /usr/sbin/policy-rc.d',
			'chmod a+x /usr/sbin/policy-rc.d',

			# Add pkg repo config
			# Not doing this here any more, this is done in debootstrap builder since it is the one
			# who knows which distro/flavor was built and can populate these correcrtly.
			#"echo deb #{mirror_url} trusty          main restricted universe >  /etc/apt/sources.list",
			#"echo deb #{mirror_url} trusty-updates  main restricted universe >> /etc/apt/sources.list",
			#"echo deb #{mirror_url} trusty-security main restricted universe >> /etc/apt/sources.list",

			# Update the apt cache
			'apt-get update',

			# Get latest upgrades
			upgrade ? 'apt-get --yes upgrade' : '',

			# Additional packages
			"apt-get #{common_apt_opts} install #{ADDITIONAL_PACKAGES.join(' ')}",

			# Download additional developer packages
			dev_build ? "apt-get #{common_apt_opts} install #{DEV_BUILD_PKGS.join(' ')}" : '',

			# Download and unpack prometheus
			"mkdir /opt/prometheus",
			"wget #{PROMETHEUS_URL} -q -O - | tar -xzf - -C /opt/prometheus --strip=1",
			"wget #{PROMETHEUS_EXPORTER_URL} -q -O - | tar -xzf - -C /usr/local/bin",

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

		info('Moving parts from warehouse into target')
		# We need to perform the copy as root, since the dest dir is owned by root.
		execute!("cp -r #{File.join(ROCKETSHIP_ROOTFS_DIR_PATH, '.')} #{rootfs_dir}")

		nil
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

		info('Packaging in tar format')
		execute!(cmd, true)

		nil
	end
end
