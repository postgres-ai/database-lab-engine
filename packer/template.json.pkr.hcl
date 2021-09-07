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
    name                               = "*ubuntu-focal-20.04-amd64-server-*"
    root-device-type                   = "ebs"
    virtualization-type                = "hvm"
  }
  most_recent = true
  owners      = ["099720109477"]
}

source "amazon-ebs" "base" {
  ami_description = "Installed AMI with Ubuntu 20.04, ZFS, Docker, Envoy proxy and Database Lab Engine 2.0 with client CLI."
  ami_name        = "${var.ami_name_prefix}-${var.dle_version}-${formatdate("YYYY-MM-DD", timestamp())}-${uuidv4()}"
  instance_type   = "t2.large"
  source_ami      = "${data.amazon-ami.base.id}"
  ssh_username    = "ubuntu"
  ami_groups      = ["all"] # This makes our AMI public
  ami_regions     = ["us-east-1","us-east-2","us-west-1","us-west-2","eu-central-1","eu-north-1","ap-south-1","eu-west-3",
                      "eu-west-2","eu-west-1","ap-northeast-3","ap-northeast-2","ap-northeast-1","sa-east-1","ca-central-1",
                    "ap-southeast-1","ap-southeast-2"] 
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

  provisioner "file"{
    source = "joe.yml"
    destination = "/home/ubuntu/joe.yml"
  }

  provisioner "shell" {
    environment_vars = ["dle_version=${var.dle_version}"]
    scripts = ["${path.root}/install-prereqs.sh", "${path.root}/install-envoy.sh"] 
  }

}
