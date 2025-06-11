#!/bin/bash

mkdir linker-upgrade
cd linker-upgrade/
curl -LO https://github.com/linker-bot/linker-upgrader/releases/download/v0.0.3/linker-upgrader_0.0.3_linux_arm64.tar.gz
tar zxvf linker-upgrader_0.0.3_linux_arm64.tar.gz

sudo curl -L https://raw.githubusercontent.com/linker-bot/linker-upgrader/refs/heads/main/scripts/etc/systemd/system/linker-upgrade.service -o /etc/systemd/system/linker-upgrade.service
sudo mkdir -p /etc/linker-upgrader/
sudo curl -L https://raw.githubusercontent.com/linker-bot/linker-upgrader/refs/heads/main/scripts/etc/linker-upgrader/config.json -o /etc/linker-upgrader/config.json

sudo mkdir -p /opt/linker-upgrader/
sudo cp ./linker-upgrader /opt/linker-upgrader/
sudo systemctl daemon-reload
sudo systemctl start linker-upgrade.service 
sudo systemctl status linker-upgrade.service 

sudo mkdir -p /opt/linkerbot
sudo mkdir -p /opt/linkerbot/backup
sudo chown -R `whoami` /opt/linkerbot/
