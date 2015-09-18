#!/bin/bash

# Needed for ruby/gem installation
export LANG=en_US.UTF-8

# Which programs to install
declare INSTALLATION_CMDS=(
    "go"
    "gcc"
    "ruby"
    "git"
    "rake"
    "bundler"
)

# How to install programs
declare -A INSTALLATION_HELP=(
    ["go"]="instructions at https://golang.org/dl"
    ["gcc"]="sudo apt-get install build-essential"
    ["ruby"]="sudo apt-get install ruby"
    ["git"]="sudo apt-get install git"
    ["rake"]="sudo gem install rake"
    ["bundler"]="sudo gem install bundler"
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

    print_green "Checking for '${cmd}':\t[$status]"

    if [ $status == "FAIL" ]; then
	print_red "Install using: ${INSTALLATION_HELP[$cmd]}"
    fi

done
