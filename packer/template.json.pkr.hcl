
variable "ami_name_prefix" {
  type    = string
  default = "${env("AMI_NAME_PREFIX")}"
}

data "amazon-ami" "base" {
  filters = {
    architecture                       = "x86_64"
    "block-device-mapping.volume-type" = "gp2"
    name                               = "*ubuntu-bionic-18.04-amd64-server-*"
    root-device-type                   = "ebs"
    virtualization-type                = "hvm"
  }
  most_recent = true
  owners      = ["099720109477"]
}

source "amazon-ebs" "base" {
  ami_description = "Installed AMI with Ubuntu 18.04, ZFS, Docker, and Database Lab Engine 2.0 with client CLI."
  ami_name        = "${var.ami_name_prefix}-${formatdate("YYYY-MM-DD", timestamp())}-${uuidv4()}"
  instance_type   = "t2.micro"
  source_ami      = "${data.amazon-ami.base.id}"
  ssh_username    = "ubuntu"
}

build {
  sources = ["source.amazon-ebs.base"]

  provisioner "shell" {
    inline = ["echo 'Sleeping for 45 seconds to give Ubuntu enough time to initialize (otherwise, packages might fail to install).'", "sleep 45", "sudo apt-get update"]
  }

  provisioner "shell" {
    scripts = ["${path.root}/install-prereqs.sh", "${path.root}/install-dblabcli.sh"]
  }

}
