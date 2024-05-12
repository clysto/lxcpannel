#!/bin/bash

# Build the self-extracting installer

go build

service=$(cat scripts/lxcpanel.service | base64 -w 0)
script=$(cat scripts/create_admin.sh | base64 -w 0)
content=$(cat <<EOF
#!/bin/bash
set -e
SERVICE="$service"
SCRIPT="$script"
BINARY=\$(awk '/^____BINARY____/ {print NR + 1; exit 0; }' \$0)
tail -n+\$BINARY \$0 > /usr/local/bin/lxcpanel
chmod +x /usr/local/bin/lxcpanel
mkdir -p /var/lib/lxcpanel
echo "\$SCRIPT" | base64 -d > /var/lib/lxcpanel/create_admin.sh
chmod +x /var/lib/lxcpanel/create_admin.sh
if [ ! -f /usr/local/lib/systemd/system/lxcpanel.service ]; then
    mkdir -p /usr/local/lib/systemd/system
    echo "\$SERVICE" | base64 -d > /usr/local/lib/systemd/system/lxcpanel.service
fi
exit 0
____BINARY____
EOF
)

echo "$content" > lxcpanel_installer.run
cat lxcpanel >> lxcpanel_installer.run
chmod +x lxcpanel_installer.run
