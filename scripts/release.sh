#!/bin/bash

# FTP server details
FTP_SERVER="home.ustc.edu.cn"
read -p "Enter your FTP username: " FTP_USERNAME
read -s -p "Enter your FTP password: " FTP_PASSWORD

# Local file path
LOCAL_FILE="lxcpanel_installer.sh"

# Remote file path
REMOTE_FILE="/public_html/archive/lxcpanel_installer.run"

# Upload file using FTP
ftp -n $FTP_SERVER <<END_SCRIPT
quote USER $FTP_USERNAME
quote PASS $FTP_PASSWORD
binary
put $LOCAL_FILE $REMOTE_FILE
quit
END_SCRIPT

if [ $? -eq 0 ]; then
    echo "File uploaded successfully"
    rm $LOCAL_FILE
else
    echo "File upload failed"
fi