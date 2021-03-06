require 'open3'
require 'pp'

require_relative 'base_builder'

class DebootstrapBuilder < BaseBuilder

	# These packages go into the barebones linux rootfs
	ESSENTIAL_PKGS = [
		'linux-image-amd64'  ,
		'dbus'               ,
		'iputils-ping'       , # ping
		'isc-dhcp-client'    , # dhcp
		'logrotate'          ,
		'net-tools'          , # ifconfig
		'rsyslog'            ,
		'openssh-server'     ,
		'wget'               ,
	]

	UBUNTU_APT_ARCHIVE_URL = "http://archive.ubuntu.com/ubuntu"
	DEBIAN_APT_ARCHIVE_URL = "http://debian.osuosl.org/debian"

	CWD = File.dirname(__FILE__)
	BUILD_DIR_PATH = File.expand_path(File.join(CWD, '..'))
	CACHE_DIR_PATH = File.expand_path(File.join(BUILD_DIR_PATH, "cache"))

	CACHED_DEBOOTSTRAP_PKGS_NAME = "debootstrap_pkgs.tgz"
	CACHED_DEBOOTSTRAP_PKGS_PATH = File.join(CACHE_DIR_PATH, CACHED_DEBOOTSTRAP_PKGS_NAME)

	DEBOOTSTRAP_ROOTFS_NAME = "debootstrap_rootfs.tgz"
	DEBOOTSTRAP_ROOTFS_PATH = File.join(CACHE_DIR_PATH, DEBOOTSTRAP_ROOTFS_NAME)

	attr_reader :verbose

	def initialize(distro, verbose)
		@distro  = distro
		@verbose = !!verbose

		case distro
		when "ubuntu"
			@flavor = "trusty"
			@archive_url = UBUNTU_APT_ARCHIVE_URL
		when "debian"
			@flavor = "jessie"
			@archive_url = DEBIAN_APT_ARCHIVE_URL
		else
			raise ArgumentError, "Invalid distro specified"
		end
	end

	def create_debootstrap_rootfs()
		header("Creating basic rootfs using debootstrap")

		notice("Ensure cache dir")
		execute!("mkdir -p #{CACHE_DIR_PATH}")

		if File.exists?(CACHED_DEBOOTSTRAP_PKGS_PATH)
			cached_pkgs_opt = "--unpack-tarball=#{CACHED_DEBOOTSTRAP_PKGS_PATH}"
			info("Cached debootstrap packages found in tarball at: #{CACHED_DEBOOTSTRAP_PKGS_PATH}")
		else
			cached_pkgs_opt = ""
			info("No cached debootstrap packages found.")
		end

		self.on_mounted_tmpfs do |tempdir|

			notice('Running debootstrap')
			execute!(["debootstrap",
				verbose ? "--verbose" : "",
				"--variant minbase",
				cached_pkgs_opt,
				"--include #{ESSENTIAL_PKGS.join(",")}",
				@flavor,
				tempdir,
				@archive_url,
			].join(" "))

			add_apt_sources(tempdir)

			notice('Packaging rootfs')
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

	def add_apt_sources(tempdir)
		notice("Adding appropriate apt sources")
		case @distro
		when "ubuntu"
			lines = [
				"deb #{@archive_url} #{@flavor}          main restricted universe",
				"deb #{@archive_url} #{@flavor}-updates  main restricted universe",
				"deb #{@archive_url} #{@flavor}-security main restricted universe",
			].join("\n")
		when "debian"
			lines = [
				"deb http://ftp.debian.org/debian #{@flavor}         main contrib",
				"deb http://ftp.debian.org/debian #{@flavor}-updates main contrib",
				"deb http://security.debian.org/  #{@flavor}/updates main contrib",
			].join("\n")
		else
			raise ArgumentError, "Unknown flavor"
		end

		execute!("echo \"#{lines}\" | sudo tee #{tempdir}/etc/apt/sources.list")
	end

	##
	# Create a debootstrap compatible tarball of deb packages.
	#
	def create_debootstrap_packages_tarball()
		header("(Re)creating tarball of packages needed for debootstrap rootfs")
		cached_pkgs_tarball = CACHED_DEBOOTSTRAP_PKGS_PATH

		notice("Ensuring old packages tarball does not exist")
		execute!("rm -f #{cached_pkgs_tarball}")

		notice("Ensure cache dir")
		execute!("mkdir -p #{CACHE_DIR_PATH}")

		self.on_mounted_tmpfs do |tempdir|
                        # create a work dir in the tempdir, because debootstrap wants to delete its work dir when
                        # it finishes, but the tempdir is owned by root.

                        workdir = File.join(tempdir, "work")
                        execute!("mkdir -p #{workdir}")

			notice("Invoking debootstrap to create new cached packages tarball")
			execute!(["debootstrap",
				verbose ? "--verbose" : "",
				"--variant minbase",
				"--include #{ESSENTIAL_PKGS.join(",")}",
				"--make-tarball #{cached_pkgs_tarball}",
				@flavor,
				workdir,
				@archive_url,
			].join(" "))
		end

		notice("debootstrap packages cached at:" + cached_pkgs_tarball)
	end

end
