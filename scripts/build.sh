#!/bin/bash

go build -o release/lxcpanel
cp scripts/setup.sh release/setup.sh
cp scripts/create_admin.sh release/create_admin.sh
cp scripts/lxcpanel.service release/lxcpanel.service

makeself release lxcpanel_installer.run lxcpanel_installer ./setup.sh
