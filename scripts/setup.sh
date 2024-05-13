#!/bin/sh

set -e

install -v -m 755 lxcpanel /usr/local/bin/lxcpanel
install -v -m 755 create_admin.sh /var/lib/lxcpanel/create_admin.sh

if [ ! -f /usr/local/lib/systemd/system/lxcpanel.service ]; then
    install -v -m 644 lxcpanel.service /usr/local/lib/systemd/system/lxcpanel.service
fi
