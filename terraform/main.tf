provider "aws" {
  access_key = var.access_key
  secret_key = var.secret_key
  region     = var.region
}

module "ecs_base" {
  source = "github.com/schramm-famm/bespin//modules/ecs_base"
  name   = var.name
}

module "ecs_cluster" {
  source                  = "github.com/schramm-famm/bespin//modules/ecs_cluster"
  name                    = var.name
  security_group_ids      = [module.ecs_base.vpc_default_security_group_id]
  subnets                 = module.ecs_base.vpc_public_subnets
  ec2_instance_profile_id = module.ecs_base.ecs_instance_profile_id
}

module "heimdall" {
  source        = "./modules/heimdall"
  name          = var.name
  container_tag = "1.0.0"
  cluster_id    = module.ecs_cluster.cluster_id
}
