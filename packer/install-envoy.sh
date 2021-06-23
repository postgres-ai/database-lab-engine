#/!bin/bash

sudo mkdir -p /etc/envoy/certs

sudo chown root.root /home/ubuntu/envoy.service
sudo mv /home/ubuntu/envoy.service /etc/systemd/system/envoy.service
sudo chown root.root /home/ubuntu/envoy.yaml
sudo mv /home/ubuntu/envoy.yaml /etc/envoy/envoy.yaml

