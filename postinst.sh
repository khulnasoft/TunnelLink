#!/bin/bash
set -eu
ln -s /usr/bin/tunnellink /usr/local/bin/tunnellink
mkdir -p /usr/local/etc/tunnellink/
touch /usr/local/etc/tunnellink/.installedFromPackageManager || true
