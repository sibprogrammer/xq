#!/bin/bash -xv

set -e

[ "$HOOK_TARGET" != "darwin_arm64" ] && upx --best --lzma $HOOK_PATH

exit 0
