variable "ami_name_prefix" {
  type    = string
  default = "${env("AMI_NAME_PREFIX")}"
}

variable "dle_version" {
  type    = string
  default = "${env("DLE_VERSION")}"
}

data "amazon-ami" "base" {
  filters = {
    architecture                       = "x86_64"
    "block-device-mapping.volume-type" = "gp2"
    name                               = "*ubuntu/images/hvm-ssd/ubuntu-focal-20.04-amd64-server-*"
    root-device-type                   = "ebs"
    virtualization-type                = "hvm"
  }
  most_recent = true
  owners      = ["099720109477"]
}

source "amazon-ebs" "base" {
  ami_description = "Ubuntu 20.04 with  ZFS, Docker, Envoy proxy and Database Lab Engine 3.1 with client CLI."
  ami_name        = "${var.ami_name_prefix}-${var.dle_version}-${formatdate("YYYY-MM-DD", timestamp())}-${uuidv4()}"
  instance_type   = "t2.large"
  source_ami      = "${data.amazon-ami.base.id}"
  ssh_username    = "ubuntu"
  ami_groups      = ["all"] # This makes our AMI public
  ami_regions     = ["eu-central-1"]
}

build {
  sources = ["source.amazon-ebs.base"]

  provisioner "shell" {
    inline = ["echo 'Sleeping for 45 seconds to give Ubuntu enough time to initialize (otherwise, packages might fail to install).'", "sleep 45", "sudo apt-get update"]
  }

  provisioner "file"{
    source = "envoy.service"
    destination = "/home/ubuntu/envoy.service"
  }
  provisioner "file"{
    source = "envoy.yaml"
    destination = "/home/ubuntu/envoy.yaml"
  }

  provisioner "shell" {
    environment_vars = ["dle_version=${var.dle_version}"]
    scripts = ["${path.root}/install-prereqs.sh", "${path.root}/install-envoy.sh","${path.root}/install-dblabcli.sh"] 
  }

  provisioner "shell" {
    inline = ["echo 'remove authorized keys'", "sudo rm /home/ubuntu/.ssh/authorized_keys", "sudo rm /root/.ssh/authorized_keys"]
  }
}
