require 'open3'
require 'pp'
require 'ostruct'

require_relative 'base_builder'

#- - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
# Build disks using images
#
class DiskBuilder < BaseBuilder

	#
	# Constants for the disk build
	#
	PARTITION_TABLE_TYPE      = 'msdos'
	FS_TYPE                   = 'ext4'

	GRUB_ARCHITECTURE         = 'i386-pc' # TODO: infer this?
	GRUB_HIDDEN_TIMEOUT       = 5
	GRUB_MENU_TIMEOUT         = 10

	GRUB_PARTITION_LABEL      = 'GRUB'
	CONFIG_PARTITION_LABEL    = 'CONFIG'
	CONFIG_PARTITION_MOUNT    = '/config'

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
	attr_reader :debug

	def initialize(dev, image_path, opts={})
		if dev.nil?
			raise ArgumentError, 'No target device (disk) specified!'
		elsif not File.blockdev?(dev)
			raise ArgumentError, "No such device found! (#{dev})"
		elsif not File.exists?(image_path)
			raise ArgumentError, 'No image available for installation'
		end

		@dev = dev
		@image_tarball_path = image_path

		@debug = opts[:debug]
	end

	##
	# Image the disk.
	#
	def build
		self.ensure_root_privilege

		banner("Installing image (#{image_tarball_path}) on to #{dev}")

		self.ensure_device_unmounted
		self.create_partitions
		self.install_grub
		self.install_system_image

		info("Device #{dev} has been prepared as system disk")
	end

	##
	# Ensure that specified device isn't already mounted
	#
	def ensure_device_unmounted
		# NOTE: execute! prints output to console, and its not necessary here.
		lines, _, status = Open3.capture3("mount | grep #{dev}")

		return unless status.success? # grep returns success if pattern is found

		lines.split("\n").each do |line|
			mounted_partition = line.split(' ')[0]
			execute!("umount #{mounted_partition}")
		end
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
			execute!("mkfs.#{FS_TYPE} -L \"#{part.label}\" #{dev}#{index+1}")

			# calculate start for next iteration
			start_size = "#{end_size}MB"
		end

		nil
	end

	##
	# Install the grub bootloader.
	#
	def install_grub
		# mount it at some temp location, and operate on it
		Dir.mktmpdir do |mountdir|
			begin
				grub_part = File.join('/dev/disk/by-label', GRUB_PARTITION_LABEL)
				execute!("mount #{grub_part} #{mountdir}")

				boot_dir = File.join(mountdir, 'boot')
				grub_dir = File.join(boot_dir, 'grub')

				Dir.mkdir(boot_dir) rescue Errno::EEXIST
				Dir.mkdir(grub_dir) rescue Errno::EEXIST

				# Copy grub files
				if not Dir.exists?("/usr/lib/grub/#{GRUB_ARCHITECTURE}")
					raise RuntimeError, 'Cannot perform GRUB installation without the '\
					"necessary files (Missing: #{"/usr/lib/grub/#{GRUB_ARCHITECTURE}"})"
				else
					FileUtils.cp_r(Dir.glob("/usr/lib/grub/#{GRUB_ARCHITECTURE}/*"),
					grub_dir)
				end

				device_map_filepath = File.join(grub_dir, 'device.map')
				load_cfg_filepath   = File.join(grub_dir, 'load.cfg')
				grub_cfg_filepath   = File.join(grub_dir, 'grub.cfg')

				core_img_filepath   = File.join(grub_dir, 'core.img')
				boot_img_filepath   = File.join(grub_dir, 'boot.img')

				# Setup device.map
				File.open(device_map_filepath, 'w') do |f|
					f.puts("(hd0) #{dev}")
				end

				# Setup load.cfg
				File.open(load_cfg_filepath, 'w') do |f|
					f.puts("search.fs_label #{GRUB_PARTITION_LABEL} root")
					f.puts('set prefix=($root)/boot/grub')
					f.puts('')
				end

				# Setup grub.cfg
				File.open(grub_cfg_filepath, 'w') do |f|
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

						k_cmdline_opts = ['rw',
							debug ? '--debug' : '',
							'console=ttyS0',
							'console=tty0'
						].join(' ') # TODO: quiet splash
						menuentry_name = "AKSHAY-#{label}" # TODO

						f.puts ('# 0')
						f.puts("menuentry \"#{menuentry_name}\" {") # TODO (ver) name?
						#f.puts('  insmod ext2') # also does ext{2,3,4}
						#f.puts('  insmod gzio')
						#f.puts('  insmod part_msdos')
						f.puts("  search  --label --set=root --no-floppy #{label}")
						f.puts("  linux   /vmlinuz root=LABEL=#{label} #{k_cmdline_opts}")
						f.puts("  initrd  /initrd.img")
						f.puts('}')
						f.puts('')
					end
				end

				# create core.img
				execute!([ 'grub-mkimage'                 ,
					"--config=#{load_cfg_filepath}",
					"--output=#{core_img_filepath}",
					# Different prefix command (unlike load.cfg)
					"--prefix=\"/boot/grub\""      ,
					'--format=i386-pc'             ,
					# TODO msdospart? also ext2 covers ext3,4
					"biosdisk ext2 part_msdos search" ,
				].join(' '))

				unless File.exists?(core_img_filepath)
					raise RuntimeError, 'No file output from grub-mkimage'
				end

				execute!([ 'grub-setup '              ,
					"--boot-image=boot.img"    , # TODO
					"--core-image=core.img"    ,
					"--directory=#{grub_dir} " ,
					"--device-map=#{device_map_filepath} " ,
					debug ? '--verbose' : ''   ,
					'--skip-fs-probe'          ,
					"#{dev}"                   ,
				].join(' '))

			rescue => e
				puts e
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
					execute!("mount #{File.join('/dev/disk/by-label', label.to_s)} "\
					"#{mountdir}")

					execute!([ 'tar ',
						'--extract',
						"--file=#{image_tarball_path}",
						# Perms from the image should be retained.
						# Our job is to only install image to disk.
						'--preserve-permissions',
						'--numeric-owner',
						"-C #{mountdir} ."].join(' '))

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
					File.open(File.join(mountdir, '/etc/fstab'), 'w') do |f|
						f.puts('# This file is autogenerated')
						f.puts(fstab_contents)
					end

				rescue => e
					puts e
				ensure
					execute!("umount #{mountdir}")
				end
			end
		end

		execute!('sync')

		nil
	end

end
