#!/bin/bash

# Needed for ruby/gem installation
export LANG=en_US.UTF-8

# Which programs need to be installed
declare INSTALLATION_CMDS=(
    "debootstrap"
    "go"
    "gcc"
    "ruby"
    "git"
    "rake"
    "tar"
    "chroot"
    "parted"
    "losetup"
    "grub-mkimage"
    "grub-bios-setup"
    "qemu-img"
)

# How to install missing programs
declare -A INSTALLATION_HELP=(
    ["debootstrap"]="sudo apt-get install debootstrap"
    ["go"]="instructions at https://golang.org/dl"
    ["gcc"]="sudo apt-get install build-essential"
    ["ruby"]="sudo apt-get install ruby1.9.1"
    ["git"]="sudo apt-get install git"
    ["rake"]="sudo gem install rake"
    ["tar"]="sudo apt-get install tar"
    ["chroot"]="sudo apt-get install coreutils"
    ["parted"]="sudo apt-get install parted"
    ["losetup"]="sudo apt-get install mount"
    ["grub-mkimage"]="sudo apt-get install grub-common"
    ["grub-bios-setup"]="sudo apt-get install grub-pc"
    ["qemu-img"]="sudo apt-get install qemu-utils"
)

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

#
# Print in red
#
function print_red() {
    echo -e "${RED}$1${NC}"
}

#
# Print in red
#
function print_green() {
    echo -e "${GREEN}$1${NC}"
}

#
# test if command exists, returns "OK" or "FAIL"
#
function check_program_exists() {
    cmd=$1

    type -P $cmd 1>/dev/null 2>&1

    if [ $? -eq 0 ]; then
	status="OK"
    else
	status="FAIL"
    fi

    echo $status
}

#
# main
#
for cmd in ${INSTALLATION_CMDS[@]}; do
    status=$(check_program_exists $cmd)

    line=$(printf '%-50s [%s]' "Checking for $cmd" "$status")
    print_green "$line"

    if [ "$status" == "FAIL" ]; then
	print_red "Install using: ${INSTALLATION_HELP[$cmd]}"
    fi

done

if [ "x$GOPATH" == "x" ]; then
        print_red "No GOPATH detected!"
        print_red "Once 'go' is installed, check out this source code in your GOPATH"
        exit 2
fi
