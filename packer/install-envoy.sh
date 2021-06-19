#/!bin/bash

sudo mkdir -p /etc/envoy/certs

sudo chown root.root /home/ubuntu/envoy.service
sudo mv /home/ubuntu/envoy.service /etc/systemd/system/envoy.service
sudo chown root.root /home/ubuntu/envoy.yaml
sudo mv /home/ubuntu/envoy.yaml /etc/envoy/envoy.yaml
sudo chown root.root /home/ubuntu/example.com.key
sudo mv /home/ubuntu/example.com.key /etc/envoy/certs/
sudo chown root.root /home/ubuntu/postgres.example.com.csr
sudo mv /home/ubuntu/postgres.example.com.csr /etc/envoy/certs/

sudo systemctl enable envoy
sudo systemctl start envoy

