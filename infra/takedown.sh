tofu destroy -target=module.ec2_east.aws_instance.instance -auto-approve
tofu destroy -target=module.ec2_mid.aws_instance.instance -auto-approve
tofu destroy -target=module.ec2_west.aws_instance.instance -auto-approve

