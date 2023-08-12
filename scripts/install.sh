#!/bin/bash

set -e

URL_PREFIX="https://github.com/sibprogrammer/xq"
INSTALL_DIR=/usr/local/bin/
BINARY=xq
LATEST_VERSION=$(curl -L -s -H 'Accept: application/json' $URL_PREFIX/releases/latest | sed -e 's/.*"tag_name":"v\([^"]*\)".*/\1/')
PLATFORM=$(uname -s | tr A-Z a-z)
case "$(uname -m)" in
  arm64)
    ARCH=arm64
    ;;
  armv6l)
    ARCH=armv6
    ;;
  armv7l)
    ARCH=armv7
    ;;
  *)
    ARCH=amd64
    ;;
esac
ARCHIVE="${BINARY}_${LATEST_VERSION}_${PLATFORM}_${ARCH}.tar.gz"
URL="$URL_PREFIX/releases/download/v${LATEST_VERSION}/$ARCHIVE"

echo "Installation of $BINARY"
rm -f $INSTALL_DIR$BINARY
curl -sSL "$URL" | tar xz -C $INSTALL_DIR $BINARY
chmod +x $INSTALL_DIR$BINARY
echo "Successfully installed at $INSTALL_DIR$BINARY"
