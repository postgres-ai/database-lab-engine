resource "aws_instance" "aws_ec2" {
  ami             = "${data.aws_ami.ami.id}"
  instance_type   = "${var.instance_type}"
  security_groups = ["${aws_security_group.sg.name}"]
  key_name = "${var.keypair}"
  tags = "${local.common_tags}"
}
