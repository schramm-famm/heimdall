provider "aws" {
  access_key = var.access_key
  secret_key = var.secret_key
  region     = var.region
}

module "ecs_base" {
  source = "github.com/schramm-famm/bespin//modules/ecs_base"
  name   = var.name
}

resource "aws_security_group" "heimdall" {
  name        = "${var.name}_allow_testing"
  description = "Allow traffic necessary for integration testing"
  vpc_id      = module.ecs_base.vpc_id

  ingress {
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = -1
    cidr_blocks = ["0.0.0.0/0"]
  }
}

module "ecs_cluster" {
  source                  = "github.com/schramm-famm/bespin//modules/ecs_cluster"
  name                    = var.name
  security_group_ids      = [aws_security_group.heimdall.id]
  subnets                 = module.ecs_base.vpc_public_subnets
  ec2_instance_profile_id = module.ecs_base.ecs_instance_profile_id
}

module "heimdall" {
  source           = "./modules/heimdall"
  name             = var.name
  container_tag    = var.container_tag
  cluster_id       = module.ecs_cluster.cluster_id
  vpc_id           = module.ecs_base.vpc_id
  security_groups  = [aws_security_group.heimdall.id]
  subnets          = module.ecs_base.vpc_public_subnets
  private_key_cert = var.private_key_cert
  cert             = var.cert
}
