#!/bin/sh

# Copyright 2014 Rafael Dantas Justo. All rights reserved.
# Use of this source code is governed by a GPL
# license that can be found in the LICENSE file.

pack_name="shelter"
version="0.1"
vendor="Rafael Dantas Justo"
maintainer="Rafael Dantas Justo <adm@rafael.net.br>"
url="http://github.com/rafaeljusto/shelter"
license="GPL"
description="System that checks periodically DNS servers for DNS and DNSSEC misconfigurations"

# https://www.debian.org/doc/debian-policy/ch-opersys.html - section 9.1.2
#
# As mandated by the FHS, packages must not place any files in /usr/local, either by putting
# them in the file system archive to be unpacked by dpkg or by manipulating them in their
# maintainer scripts.
#
# However, the package may create empty directories below /usr/local so that the system
# administrator knows where to place site-specific files. These are not directories in
# /usr/local, but are children of directories in /usr/local. These directories
# (/usr/local/*/dir/) should be removed on package removal if they are empty.

install_path=/usr/shelter
tmp_dir=/tmp/shelter
project_root=$tmp_dir$install_path

workspace=`echo $GOPATH | cut -d: -f1`
workspace=$workspace/src/github.com/rafaeljusto/shelter

# recompiling everything
current_dir=`pwd`
cd $workspace
go build shelter.go
cd $workspace/deploy/debian
go build config_init.go
go build generate_cert.go
cd $current_dir

if [ -f $pack_name*.deb ]; then
  # remove old deb
  rm $pack_name*.deb
fi

if [ -d $tmp_dir ]; then
  rm -rf $tmp_dir
fi

mkdir -p $tmp_dir$install_path
mkdir -p $tmp_dir$install_path/bin
mkdir -p $tmp_dir$install_path/var/log

cp -r $workspace/etc $project_root/
cp -r $workspace/templates $project_root/
mv $workspace/shelter $project_root/bin/
mv $workspace/deploy/debian/config_init $project_root/bin/
mv $workspace/deploy/debian/generate_cert $project_root/bin/

fpm -s dir -t deb \
  --after-install $project_root/bin/config_init \
  --exclude=.git -n $pack_name -v $version --vendor "$vendor" \
  --maintainer "$maintainer" --url $url --license $license --description "$description" \
  --deb-upstart $workspace/deploy/debian/shelter.upstart \
  --deb-user root --deb-group root \
  --prefix / -C $tmp_dir usr/shelter
