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


module "ec2_east" {
  source = "./ec2_module"
  ami = "ami-07d9b9ddc6cd8dd30"
  name = "east"
  providers = {
    aws = aws.east
  }
  route_zone_id = "Z00325221D3WV6IM3G34A"
}
// module "ec2_mid" {
//   source = "./ec2_module"
//   ami = "ami-0f5daaa3a7fb3378b"
//   name = "mid"
//   providers = {
//     aws = aws.mid
//   }
//   route_zone_id = "Z00325221D3WV6IM3G34A"
// }
// module "ec2_west" {
//   source = "./ec2_module"
//   ami = "ami-0a0409af1cb831414"
//   name = "west"
//   providers = {
//     aws = aws.west
//   }
//   route_zone_id = "Z00325221D3WV6IM3G34A"
// }




