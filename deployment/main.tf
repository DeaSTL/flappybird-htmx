provider "aws" {
  alias = "east"
  region = "us-east-1"
}

provider "aws" {
  alias = "west"
  region = "us-west-1"
}

provider "aws" {
  alias = "mid"
  region = "us-east-2"
}

resource "aws_key_pair" "deployer_key_east" {
  key_name = "deployer-key"
  public_key = file("~/.ssh/id_rsa.pub")
}
resource "aws_key_pair" "deployer_key_mid" {
  key_name = "deployer-key"
  provider = aws.mid
  public_key = file("~/.ssh/id_rsa.pub")
}
resource "aws_key_pair" "deployer_key_west" {
  key_name = "deployer-key"
  provider = aws.west
  public_key = file("~/.ssh/id_rsa.pub")
}

resource "aws_instance" "east_instance" {
  count = 1
  ami = "ami-07d9b9ddc6cd8dd30"
  instance_type = "t2.nano"
  provider = aws.east
  key_name = aws_key_pair.deployer_key_east.key_name

  tags = {
    Name = "Flappy Bird East ${count.index}"
  }
}

resource "aws_instance" "mid_instance" {
  count = 1
  ami = "ami-0f5daaa3a7fb3378b"
  instance_type = "t2.nano"
  provider = aws.mid
  key_name = aws_key_pair.deployer_key_mid.key_name

  tags = {
    Name = "Flappy Bird Mid ${count.index}"
  }
}

resource "aws_instance" "west_instance" {
  count = 1
  ami = "ami-0a0409af1cb831414"
  instance_type = "t2.nano"
  provider = aws.west
  key_name = aws_key_pair.deployer_key_west.key_name

  tags = {
    Name = "Flappy Bird West ${count.index}"
  }

}

resource "aws_route53_record" "route_east" {
  count = length(aws_instance.east_instance.*.public_ip)
  zone_id = "Z00325221D3WV6IM3G34A"
  name = "instance-${count.index}.east_instance.com"
  type = "A"
  ttl = "300"
  records = [aws_instance.east_instance[count.index].public_ip]
}
resource "aws_route53_record" "route_west" {
  count = length(aws_instance.west_instance.*.public_ip)
  zone_id = "Z00325221D3WV6IM3G34A"
  name = "instance-${count.index}.west_instance.com"
  type = "A"
  ttl = "300"
  records = [aws_instance.west_instance[count.index].public_ip]
}
resource "aws_route53_record" "route_mid" {
  count = length(aws_instance.mid_instance.*.public_ip)
  zone_id = "Z00325221D3WV6IM3G34A"
  name = "instance-${count.index}.mid_instance.com"
  type = "A"
  ttl = "300"
  records = [aws_instance.mid_instance[count.index].public_ip]
}

