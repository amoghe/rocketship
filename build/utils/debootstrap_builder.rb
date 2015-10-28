require 'open3'
require 'pp'

require_relative 'base_builder'

class DebootstrapBuilder < BaseBuilder

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

	UBUNTU_APT_ARCHIVE_URL = "http://archive.ubuntu.com/ubuntu"

	CWD = File.dirname(__FILE__)
	BUILD_DIR_PATH = File.expand_path(File.join(CWD, '..'))
	CACHE_DIR_PATH = File.expand_path(File.join(BUILD_DIR_PATH, "cache"))

	CACHED_DEBOOTSTRAP_PKGS_NAME = "debootstrap_pkgs.tgz"
	CACHED_DEBOOTSTRAP_PKGS_PATH = File.join(CACHE_DIR_PATH, CACHED_DEBOOTSTRAP_PKGS_NAME)

	DEBOOTSTRAP_ROOTFS_NAME = "debootstrap_rootfs.tar.gz"
	DEBOOTSTRAP_ROOTFS_PATH = File.join(CACHE_DIR_PATH, DEBOOTSTRAP_ROOTFS_NAME)

	def initialize
	end

	def create_debootstrap_rootfs()

		if File.exists?(CACHED_DEBOOTSTRAP_PKGS_PATH)
			cached_pkgs_opt = "--unpack-tarball=#{CACHED_DEBOOTSTRAP_PKGS_PATH}"
			info("Cached debootstrap packages found in tarball at: #{CACHED_DEBOOTSTRAP_PKGS_PATH}")
		else
			cached_pkgs_opt = ""
			info("No cached debootstrap packages found.")
		end

		self.on_mounted_tmpfs do |tempdir|

			info('Running debootstrap')
			execute!(["debootstrap",
				"--variant minbase",
				cached_pkgs_opt,
				"--include #{ESSENTIAL_PKGS.join(",")}",
				"trusty",
				tempdir,
				UBUNTU_APT_ARCHIVE_URL,
			].join(" "))

			cmd =

			info('Packaging rootfs')
			execute!(['tar ',
				'--create',
				'--gzip',
				"--file=#{DEBOOTSTRAP_ROOTFS_PATH}",
				# TODO: preserve perms, else whoever uses the image will have to twidle the perms again.
				#'--owner=0',
				#'--group=0',
				'--preserve-permissions',
				'--numeric-owner',
				"-C #{tempdir} ."
			].join(' '),
			true)

		end
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
