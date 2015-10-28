require 'open3'
require 'pp'
require 'ostruct'
require 'tempfile'

require_relative 'base_builder'

#- - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
# Build disks using images
#
class DiskBuilder < BaseBuilder

	#
	# Constants for the disk build
	#
	BUILD_DIR                 = File.expand_path(File.join(File.dirname(__FILE__), '..'))

	VMDK_FILE_NAME            = "rocketship.vmdk"
	VMDK_FILE_PATH            = File.join(BUILD_DIR, VMDK_FILE_NAME)

	PARTITION_TABLE_TYPE      = 'msdos'
	FS_TYPE                   = 'ext4'

	GRUB_ARCHITECTURE         = 'i386-pc' # TODO: infer this?
	GRUB_HIDDEN_TIMEOUT       = 5
	GRUB_MENU_TIMEOUT         = 10

	GRUB_PARTITION_LABEL      = 'GRUB'
	CONFIG_PARTITION_LABEL    = 'CONFIG'
	CONFIG_PARTITION_MOUNT    = '/config'

	TOTAL_DISK_SIZE_GB        = 8
	GRUB_PARTITION_SIZE_MB    = 64
	OS_PARTITION_SIZE_MB      = 1 * 1024 # 1 GB
	CONFIG_PARTITION_SIZE_MB  = 2 * 1024

	PARTITIONS = [
			OpenStruct.new(:type  => :grub,
				:label => GRUB_PARTITION_LABEL,
				:size  => GRUB_PARTITION_SIZE_MB),
			OpenStruct.new(:type  => :os,
				:label => 'BOOTBANK1',
				:size  => OS_PARTITION_SIZE_MB),
			OpenStruct.new(:type  => :os,
				:label => 'BOOTBANK2',
				:size  => OS_PARTITION_SIZE_MB),
			OpenStruct.new(:type  => :data,
				:label => CONFIG_PARTITION_LABEL,
				:size  => CONFIG_PARTITION_SIZE_MB),
	]

	attr_reader :dev
	attr_reader :image_tarball_path
	attr_reader :verbose

	def initialize(image_path)
		raise ArgumentError, "Invalid image specified: #{image_path}" unless File.exists?(image_path)

		@image_tarball_path = image_path
		@verbose = false

		# these will be set later
		@dev = nil
		@tempfile = nil
	end

	##
	# Image the disk.
	#
	def build
		header("Building disk")
		self.create_loopback_disk
		self.create_partitions
		self.install_grub
		self.install_system_image
		self.create_vmdk
	rescue => e
		warn("Failed to build disk due to #{e}")
		warn(e.backtrace)
	ensure
		self.delete_loopback_disk
	end

	##
	# Create the loopback disk device on which we'll first install the image
	#
	def create_loopback_disk
		banner("Creating disk file and loopback device")

		@tempfile = "/tmp/tempdisk_#{Time.now.to_i}"
		execute!("fallocate -l #{TOTAL_DISK_SIZE_GB}G #{@tempfile}", false)

		output, _, stat = Open3.capture3("sudo losetup --find")
		raise RuntimeError, 'Failed to find loop device' unless stat.success?

		execute!("losetup #{output.strip} #{@tempfile}")
		@dev = output.strip

		info("Using file  : #{@tempfile}")
		info("Using device: #{dev}")
	end

	##
	# Delete the loopback disk device
	#
	def delete_loopback_disk
		banner("Deleting loop disk and file")
		execute!("losetup -d #{dev}") if dev && dev.length > 0
		execute!("rm -f #{@tempfile}") if @tempfile && @tempfile.length > 0
	end

	##
	# Create the partitions on the disk.
	#
	def create_partitions
		#TODO: do logical partitions
		if PARTITIONS.count > 4
			raise RuntimeError, 'Cannot create more than 4 partitions'
		end

		execute!("parted -s #{dev} mklabel #{PARTITION_TABLE_TYPE}")

		start_size    = 1 # MB
		end_size      = 0 # MB

		PARTITIONS.each_with_index do |part, index|
			end_size += part.size

			info("Creating partition #{part.label} (#{FS_TYPE})")

			# create a partition
			execute!("parted #{dev} mkpart primary #{FS_TYPE} #{start_size} #{end_size}MB")

			# put a filesystem and label on it
			execute!("mkfs.#{FS_TYPE} -L \"#{part.label}\" #{dev}p#{index+1}")

			# calculate start for next iteration
			start_size = "#{end_size}MB"
		end

		nil
	end

	##
	# Install the grub bootloader.
	#
	def install_grub
		info("installing grub")
		# mount it at some temp location, and operate on it
		Dir.mktmpdir do |mountdir|
			begin
				grub_part = File.join('/dev/disk/by-label', GRUB_PARTITION_LABEL)
				execute!("mount #{grub_part} #{mountdir}")

				boot_dir = File.join(mountdir, 'boot')
				grub_dir = File.join(boot_dir, 'grub')
				arch_dir = File.join(grub_dir, GRUB_ARCHITECTURE)

				execute!("mkdir -p #{boot_dir}")
				execute!("mkdir -p #{grub_dir}")
				execute!("mkdir -p #{arch_dir}")

				# Copy grub files
				if not Dir.exists?("/usr/lib/grub/#{GRUB_ARCHITECTURE}")
					raise RuntimeError, 'Cannot perform GRUB installation without the '\
					"necessary files (Missing: #{"/usr/lib/grub/#{GRUB_ARCHITECTURE}"})"
				else
					execute!("cp -r /usr/lib/grub/#{GRUB_ARCHITECTURE} #{grub_dir}")
				end

				device_map_filepath = File.join(grub_dir, 'device.map')
				load_cfg_filepath   = File.join(grub_dir, 'load.cfg')
				grub_cfg_filepath   = File.join(grub_dir, 'grub.cfg')

				core_img_filepath   = File.join(arch_dir, 'core.img')
				boot_img_filepath   = File.join(arch_dir, 'boot.img')

				# Setup device.map
				info("creating device map")
				Tempfile.open('device.map') do |f|
					f.puts("(hd0) #{dev}")

					f.sync; f.fsync # flush ruby buffers and OS buffers
					execute!("cp #{f.path} #{device_map_filepath}")
				end

				# Setup load.cfg
				info("creating load.cfg")
				Tempfile.open('load.cfg') do |f|
					f.puts("search.fs_label #{GRUB_PARTITION_LABEL} root")
					f.puts("set prefix=($root)/boot/grub")

					f.sync; f.fsync # flush ruby buffers and OS buffers
					execute!("cp #{f.path} #{load_cfg_filepath}")
				end

				# Setup grub.cfg
				info("creating grub.cfg")
				Tempfile.open('grub.conf') do |f|
					f.puts('set default=0')  # TODO ?
					f.puts("set timeout=#{GRUB_MENU_TIMEOUT}")
					f.puts('')

					f.puts('set menu_color_normal=white/black')
					f.puts('set menu_color_highlight=black/light-gray')
					f.puts('')

					f.puts("if sleep --verbose --interruptible #{GRUB_HIDDEN_TIMEOUT} ; then")
					f.puts('  echo "Loading ..."')
					f.puts('  set timeout=0')
					f.puts('fi')

					PARTITIONS.select { |part| part.type == :os }.each do |part|

						label = part.label
						size  = part.size

						k_cmdline_opts_normal = [ 'rw', 'quiet', 'splash' ].join(' ')
						k_cmdline_opts_debug  = [ 'rw', 'debug', 'console=tty0' ].join(' ')

						# The "Normal" entry
						f.puts ('# 0')
						f.puts("menuentry \"ROCKETSHIP_#{label}\" {") # TODO (ver) name?
						f.puts('  insmod ext2') # also does ext{2,3,4}
						#f.puts('  insmod gzio')
						#f.puts('  insmod part_msdos')
						f.puts("  search  --label --set=root --no-floppy #{label}")
						f.puts("  linux   /vmlinuz root=LABEL=#{label} #{k_cmdline_opts_normal}")
						f.puts("  initrd  /initrd.img")
						f.puts('}')
						f.puts('')

						f.puts ('# 1')
						f.puts("menuentry \"ROCKETSHIP_#{label}_DEBUG\" {") # TODO (ver) name?
						f.puts('  insmod ext2') # also does ext{2,3,4}
						#f.puts('  insmod gzio')
						#f.puts('  insmod part_msdos')
						f.puts("  search  --label --set=root --no-floppy #{label}")
						f.puts("  linux   /vmlinuz root=LABEL=#{label} #{k_cmdline_opts_debug}")
						f.puts("  initrd  /initrd.img")
						f.puts('}')
					end

					f.sync; f.fsync # flush from ruby buffers, then os buffers

					# Copy it over
					execute!("cp #{f.path} #{grub_cfg_filepath}")
				end

				# create core.img
				execute!([ 'grub-mkimage'		,
					"--config=#{load_cfg_filepath}"	,
					"--output=#{core_img_filepath}"	,
					# Different prefix command (unlike load.cfg)
					"--prefix=\"/boot/grub\""	,
					"--format=#{GRUB_ARCHITECTURE}"	,
					# TODO msdospart? also ext2 covers ext3,4
					"biosdisk ext2 part_msdos search" ,
				].join(' '))

				unless File.exists?(core_img_filepath)
					raise RuntimeError, 'No file output from grub-mkimage'
				end

				execute!([
					'grub-bios-setup'          ,
					"--boot-image=#{GRUB_ARCHITECTURE}/boot.img"    ,
					"--core-image=#{GRUB_ARCHITECTURE}/core.img"    ,
					"--directory=#{grub_dir} " ,
					"--device-map=#{device_map_filepath} " ,
					verbose ? '--verbose' : ''   ,
					'--skip-fs-probe'          ,
					"#{dev}"                   ,
				].join(' '))

			ensure
				# Always unmount it
				execute!("umount #{mountdir}")
			end
		end

		nil
	end

	##
	# Put the image on the disk.
	#
	def install_system_image(num_os=1)
		unless image_tarball_path and File.exists?(image_tarball_path)
			raise RuntimeError, 'Invalid image specified'
		end

		PARTITIONS.select { |part| part.type == :os }.take(num_os).each do |part|
			label = part.label
			size  = part.size

			# mount it, put the image on it, unmount it
			Dir.mktmpdir do |mountdir|
				begin
					execute!("mount #{File.join('/dev/disk/by-label', label.to_s)} #{mountdir}")

					execute!([ 	'tar ',
							'--extract',
							"--file=#{image_tarball_path}",
							# Perms from the image should be retained.
							# Our job is to only install image to disk.
							'--preserve-permissions',
							'--numeric-owner',
							"-C #{mountdir} ."
						].join(' '))

						fsopts = "defaults,errors=remount-ro"
						clabel = CONFIG_PARTITION_LABEL
						cmntpt = CONFIG_PARTITION_MOUNT

						fstab_contents = \
						[['# <filesystem>', '<mnt>', '<type>', '<opts>', '<dump>', '<pass>'],
						["LABEL=#{label}", '/'    , FS_TYPE , fsopts  , '0'    ,  '1'     ],
						["LABEL=#{clabel}", cmntpt, FS_TYPE , fsopts  , '0'    ,  '1'     ],
					].reduce('') do |memo, line_tokens|
						memo << line_tokens.join("\t")
						memo << "\n"
						memo
					end

					# We now know which partition we reside on, so write out the fstab
					fstab_file_path = File.join(mountdir, '/etc/fstab')
					Tempfile.open('fstab') do |f|
						f.puts('# This file is autogenerated')
						f.puts(fstab_contents)

						f.sync; f.fsync # flush ruby buffers and OS buffers
						execute!("cp #{f.path} #{fstab_file_path}")
					end

				ensure
					execute!("umount #{mountdir}")
				end
			end
		end

		execute!('sync')

		nil
	end

	def create_vmdk
		execute!("losetup -d #{dev}")
		@dev = nil # nil it out, indicating we were successful in umounting it

		execute!("qemu-img convert -f raw -O vmdk #{@tempfile} #{VMDK_FILE_PATH}")

		orig_user = `whoami`.strip
		execute!("chown #{orig_user} #{VMDK_FILE_PATH}")
	end
end
