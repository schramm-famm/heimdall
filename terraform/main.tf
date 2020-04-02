provider "aws" {
  access_key = var.access_key
  secret_key = var.secret_key
  region     = var.region
}

module "ecs_base" {
  source             = "github.com/schramm-famm/bespin//modules/ecs_base"
  name               = var.name
  enable_nat_gateway = true
}

resource "aws_security_group" "backend" {
  name        = "${var.name}_backend"
  description = "Allow traffic for backend services"
  vpc_id      = module.ecs_base.vpc_id

  ingress {
    from_port   = 8080
    to_port     = 8081
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

module "backend_ecs_cluster" {
  source                  = "github.com/schramm-famm/bespin//modules/ecs_cluster"
  name                    = "${var.name}-backend"
  security_group_ids      = [aws_security_group.backend.id]
  subnets                 = module.ecs_base.vpc_private_subnets
  ec2_instance_profile_id = module.ecs_base.ecs_instance_profile_id
}

module "heimdall_ecs_cluster" {
  source                  = "github.com/schramm-famm/bespin//modules/ecs_cluster"
  name                    = "${var.name}-heimdall"
  security_group_ids      = [aws_security_group.backend.id]
  subnets                 = module.ecs_base.vpc_private_subnets
  ec2_instance_profile_id = module.ecs_base.ecs_instance_profile_id
}

module "heimdall" {
  source              = "./modules/heimdall"
  name                = var.name
  container_tag       = var.heimdall_container_tag
  cluster_id          = module.heimdall_ecs_cluster.cluster_id
  vpc_id              = module.ecs_base.vpc_id
  external_lb_subnets = module.ecs_base.vpc_public_subnets
  internal_lb_subnets = module.ecs_base.vpc_private_subnets
  private_key_cert    = var.private_key_cert
  cert                = var.cert
  endpoints = {
    "karen" = module.karen.elb_dns_name
  }
}

module "karen" {
  source          = "github.com/schramm-famm/karen//terraform/modules/karen"
  name            = var.name
  container_tag   = var.karen_container_tag
  port            = 8081
  cluster_id      = module.backend_ecs_cluster.cluster_id
  security_groups = [aws_security_group.backend.id]
  subnets         = module.ecs_base.vpc_private_subnets
  internal        = true
  db_location     = module.rds_instance.db_endpoint
  db_username     = var.rds_username
  db_password     = var.rds_password
}

module "rds_instance" {
  source          = "github.com/schramm-famm/bespin//modules/rds_instance"
  name            = var.name
  engine          = "mariadb"
  engine_version  = "10.2.21"
  port            = 3306
  master_username = var.rds_username
  master_password = var.rds_password
  vpc_id          = module.ecs_base.vpc_id
  subnet_ids      = module.ecs_base.vpc_private_subnets
}
