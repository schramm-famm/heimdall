resource "aws_cloudwatch_log_group" "heimdall" {
  name              = "${var.name}_heimdall"
  retention_in_days = 1
}

resource "aws_ecs_task_definition" "heimdall" {
  family = "${var.name}_heimdall"

  container_definitions = <<EOF
[
  {
    "name": "${var.name}_heimdall",
    "image": "343660461351.dkr.ecr.us-east-2.amazonaws.com/heimdall:${var.container_tag}",
    "cpu": 10,
    "memory": 128,
    "essential": true,
    "environment": [
      {
        "name": "PRIVATE_KEY",
        "value": "${var.private_key}"
      }
    ],
    "portMappings": [
      {
        "containerPort": 80,
        "hostPort": 80,
        "protocol": "tcp"
      }
    ]
  }
]
EOF
}

resource "aws_ecs_service" "heimdall" {
  name            = "${var.name}_heimdall"
  cluster         = var.cluster_id
  task_definition = aws_ecs_task_definition.heimdall.arn

  desired_count = 1
}
