#!/usr/bin/env bash

################################################################################################
#
# Packaging script which creates debian and RPM packages.
#
# Requirements: GOPATH must be set. 'fpm' must be on the path, and the AWS
# CLI tools must also be installed.
#
# Packaging process: to package a build, simple execute:
#
#    package.sh <version>
#
# where <version> is the desired version. If generation of a debian and RPM
# package is successful, the script will offer to tag the repo using the
# supplied version string.
#
# Copied and modified from influxdb: https://github.com/influxdb/influxdb/blob/master/package.sh
#

NIGHTLY_BUILD="1"

AWS_FILE=

INSTALL_ROOT_DIR=/usr/sbin
CONFIG_ROOT_DIR=/etc/gracevisor
UPSTART_ROOT_DIR=/etc/init

GOPATH_INSTALL=

TMP_WORK_DIR=`mktemp -d`
POST_INSTALL_PATH=`mktemp`

UPSTART_SCRIPT=init/upstart/gracevisor.conf

ARCH=`uname -i`
LICENSE=apache
URL=https://github.com/hamaxx/gracevisor
MAINTAINER=jure@hamsworld.net
VENDOR=Gracevisor
DESCRIPTION="A Process Control System Built for the Web"

BINS=(
    gracevisord
    gracevisorctl
    )

if [ -z "$FPM" ]; then
    FPM=`which fpm`
fi

do_build() {
    for b in ${BINS[*]}; do
        rm -f $GOPATH_INSTALL/bin/$b
    done
    go get -u -f -d ./...
    if [ $? -ne 0 ]; then
        echo "WARNING: failed to 'go get' packages."
    fi

    version=$1
    commit=`git rev-parse HEAD`
    if [ $? -ne 0 ]; then
        echo "Unable to retrieve current commit -- aborting"
        cleanup_exit 1
    fi

    go install -a -ldflags="-X main.version $version -X main.commit $commit" ./...
    if [ $? -ne 0 ]; then
        echo "Build failed, unable to create package -- aborting"
        cleanup_exit 1
    fi
    echo "Build completed successfully."
}

check_gopath() {
    [ -z "$GOPATH" ] && echo "GOPATH is not set." && cleanup_exit 1
    GOPATH_INSTALL=`echo $GOPATH | cut -d ':' -f 1`
    [ ! -d "$GOPATH_INSTALL" ] && echo "GOPATH_INSTALL is not a directory." && cleanup_exit 1
    echo "GOPATH ($GOPATH) looks sane, using $GOPATH_INSTALL for installation."
}

usage() {
    echo -e "$0 [<version>] [-h]\n"
    cleanup_exit $1
}

cleanup_exit() {
    rm -r $TMP_WORK_DIR
    rm $POST_INSTALL_PATH
    exit $1
}

make_dir_tree() {
    work_dir=$1
    version=$2
    mkdir -p $work_dir/$INSTALL_ROOT_DIR
    if [ $? -ne 0 ]; then
        echo "Failed to create installation directory -- aborting."
        cleanup_exit 1
    fi
    mkdir -p $work_dir/$CONFIG_ROOT_DIR
    if [ $? -ne 0 ]; then
        echo "Failed to create configuration directory -- aborting."
        cleanup_exit 1
    fi
    mkdir -p $work_dir/$UPSTART_ROOT_DIR
    if [ $? -ne 0 ]; then
        echo "Failed to create upstart directory -- aborting."
        cleanup_exit 1
    fi
}

if [ $# -ne 1 ]; then
    usage 1
elif [ $1 == "-h" ]; then
    usage 0
else
    VERSION=$1
    VERSION_UNDERSCORED=`echo "$VERSION" | tr - _`
fi

echo -e "\nStarting package process...\n"

check_gopath
do_build $VERSION
make_dir_tree $TMP_WORK_DIR $VERSION

for b in ${BINS[*]}; do
    cp $GOPATH_INSTALL/bin/$b $TMP_WORK_DIR/$INSTALL_ROOT_DIR
    if [ $? -ne 0 ]; then
        echo "Failed to copy binaries to packaging directory -- aborting."
        cleanup_exit 1
    fi
done

cp $UPSTART_SCRIPT $TMP_WORK_DIR/$UPSTART_ROOT_DIR
if [ $? -ne 0 ]; then
    echo "Failed to copy upstart script to packaging directory -- aborting."
    cleanup_exit 1
fi
echo "$UPSTART_SCRIPT copied to $TMP_WORK_DIR/$UPSTART_ROOT_DIR"

#cp $SAMPLE_CONFIGURATION $TMP_WORK_DIR/$CONFIG_ROOT_DIR/gracevisor.yaml
#if [ $? -ne 0 ]; then
#    echo "Failed to copy $SAMPLE_CONFIGURATION to packaging directory -- aborting."
#    cleanup_exit 1
#fi
touch $TMP_WORK_DIR/$CONFIG_ROOT_DIR/gracevisor.yaml

echo -n "Commence creation of $ARCH packages, version $VERSION? [Y/n] "
read response
response=`echo $response | tr 'A-Z' 'a-z'`
if [ "x$response" == "xn" ]; then
    echo "Packaging aborted."
    cleanup_exit 1
fi

if [ $ARCH == "i386" ]; then
    rpm_package=gracevisor-${VERSION}-1.i686.rpm # RPM packages use 1 for default package release.
    debian_package=gracevisor${VERSION}_i686.deb
    deb_args="-a i686"
    rpm_args="setarch i686"
elif [ $ARCH == "arm" ]; then
    rpm_package=gracevisor-${VERSION}-1.armel.rpm
    debian_package=gracevisor${VERSION}_armel.deb
else
    rpm_package=gracevisor-${VERSION}-1.x86_64.rpm
    debian_package=gracevisor${VERSION}_amd64.deb
fi

COMMON_FPM_ARGS="-C $TMP_WORK_DIR --vendor $VENDOR --url $URL --license $LICENSE --maintainer $MAINTAINER --after-install $POST_INSTALL_PATH --name gracevisor --version $VERSION --config-files $CONFIG_ROOT_DIR ."
$rpm_args $FPM -s dir -t rpm -p packages/ --description "$DESCRIPTION" $COMMON_FPM_ARGS
if [ $? -ne 0 ]; then
    echo "Failed to create RPM package -- aborting."
    cleanup_exit 1
fi
echo "RPM package created successfully."

$FPM -s dir -t deb $deb_args -p packages/ --description "$DESCRIPTION" $COMMON_FPM_ARGS
if [ $? -ne 0 ]; then
    echo "Failed to create Debian package -- aborting."
    cleanup_exit 1
fi
echo "Debian package created successfully."

$FPM -s dir -t tar --prefix gracevisor_${VERSION}_${ARCH} -p packages/gracevisor_${VERSION}_${ARCH}.tar.gz --description "$DESCRIPTION" $COMMON_FPM_ARGS
if [ $? -ne 0 ]; then
    echo "Failed to create Tar package -- aborting."
    cleanup_exit 1
fi
echo "Tar package created successfully."

echo -n "Publish packages to S3? [y/N] "
read response
response=`echo $response | tr 'A-Z' 'a-z'`

if [ "x$response" == "xy" ]; then
    echo "Publishing packages to S3."

    for filepath in `ls packages/*.{deb,rpm,gz}`; do
        filename=`basename $filepath`
        if [ -n "$NIGHTLY_BUILD" ]; then
            filename=`echo $filename | sed s/$VERSION/nightly/`
            filename=`echo $filename | sed s/$VERSION_UNDERSCORED/nightly/`
        fi
        AWS_CONFIG_FILE=$AWS_FILE aws s3 cp $filepath s3://hamax-test/$filename --acl public-read --region us-east-1
        if [ $? -ne 0 ]; then
            echo "Upload failed -- aborting".
            cleanup_exit 1
        fi
    done
else
    echo "Not publishing packages to S3."
fi

echo -e "\nPackaging process complete."
cleanup_exit 0
