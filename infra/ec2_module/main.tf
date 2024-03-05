provider "acme" {
  #server_url = "https://acme-v02.api.letsencrypt.org/directory"
  server_url = "https://acme-staging-v02.api.letsencrypt.org/directory"
}

data "aws_route53_zone" "base_domain" {
  name = "htmx-flappybird.jmhart.dev"
}

resource "tls_private_key" "private_key" {
   algorithm = "RSA"
   rsa_bits = 2048
}

resource "acme_registration" "registration" {
  account_key_pem = tls_private_key.private_key.private_key_pem
  email_address = "jhartway99@gmail.com"
}

resource "aws_key_pair" "deployer_key" {
  key_name = "deployer-key"
  public_key = file("~/.ssh/id_rsa.pub")
}

resource "aws_route53_record" "route" {
  zone_id = var.route_zone_id
  name = "${var.name}"
  type = "A"
  ttl = "300"
  records = [aws_instance.instance.public_ip]
}

resource "acme_certificate" "certificate" {
  account_key_pem           = acme_registration.registration.account_key_pem
  common_name               = data.aws_route53_zone.base_domain.name
  subject_alternative_names = ["${var.name}.${data.aws_route53_zone.base_domain.name}"]

  dns_challenge {
    provider = "route53"
    config = {
      AWS_HOSTED_ZONE_ID = var.route_zone_id
    }
  }

  depends_on = [acme_registration.registration]
}

resource "aws_instance" "instance" {
  ami = var.ami
  instance_type = "t2.nano"
  key_name = aws_key_pair.deployer_key.key_name
  user_data = templatefile("${path.module}/startup-script.sh",
  {
    private_key = nonsensitive(lookup(acme_certificate.certificate, "private_key_pem")),
    cert = lookup(acme_certificate.certificate, "certificate_pem"),
    domain = "${var.name}.${data.aws_route53_zone.base_domain.name}"
  })

  depends_on = [acme_certificate.certificate]
}

