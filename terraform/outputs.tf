output "ip" {
  value = "${aws_instance.aws_ec2.public_ip}"
}
output "ec2instance" {
  value = "${aws_instance.aws_ec2.id}"
}
