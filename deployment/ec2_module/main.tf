resource "aws_key_pair" "deployer_key" {
  key_name = "deployer-key"
  public_key = file("~/.ssh/id_rsa.pub")
}

resource "aws_instance" "instance" {
  ami = var.ami
  instance_type = "t2.nano"
  key_name = aws_key_pair.deployer_key.key_name
  user_data = <<-EOL
  #!/bin/bash -xe
  apt-get update
  apt-get install -y docker.io
  systemctl enable docker
  systemctl start docker
  mkdir /environment
  echo '${var.name}' > /environment/region
  docker run -d --network host jhartway99/htmx-flappybird
  EOL
}

resource "aws_route53_record" "route" {
  zone_id = var.route_zone_id
  name = "${var.name}"
  type = "A"
  ttl = "300"
  records = [aws_instance.instance.public_ip]
}
