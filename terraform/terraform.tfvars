ami_name_filter = "DBLABserver-*"
aws_region = "us-east-2"
instance_type = "t2.micro"
keypair = "postgres_ext_test"
allow_ssh_from_cidrs = ["0.0.0.0/0"]
tag_name = "DBLABserver-ec2instance"
