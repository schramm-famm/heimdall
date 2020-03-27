data "aws_region" "heimdall" {}

resource "aws_cloudwatch_log_group" "heimdall" {
  name = "${var.name}_heimdall"
}

resource "aws_ecs_task_definition" "heimdall" {
  family = "${var.name}_heimdall"

  container_definitions = <<EOF
[
  {
    "name": "${var.name}_heimdall",
    "image": "343660461351.dkr.ecr.us-east-2.amazonaws.com/heimdall:${var.container_tag}",
    "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
            "awslogs-group": "${aws_cloudwatch_log_group.heimdall.name}",
            "awslogs-region": "${data.aws_region.heimdall.name}",
            "awslogs-stream-prefix": "${var.name}"
        }
    },
    "cpu": 10,
    "memory": 128,
    "essential": true,
    "environment": [
      {
        "name": "PRIVATE_KEY",
        "value": "${var.private_key_jwt}"
      },
      {
        "name": "KAREN_HOST",
        "value": "localhost:80"
      }
    ],
    "portMappings": [
      {
        "containerPort": 80,
        "hostPort": 80,
        "protocol": "tcp"
      },
      {
        "containerPort": 8080,
        "hostPort": 8080,
        "protocol": "tcp"
      }
    ]
  }
]
EOF
}

resource "aws_lb" "heimdall-internal" {
  name               = "${var.name}-heimdall-internal"
  internal           = true
  load_balancer_type = "network"
  subnets            = var.subnets
}

resource "aws_lb" "heimdall-external" {
  name               = "${var.name}-heimdall-external"
  internal           = false
  load_balancer_type = "network"
  subnets            = var.subnets
}

resource "aws_lb_target_group" "heimdall-internal" {
  name        = "${var.name}-heimdall-internal"
  port        = 8080
  protocol    = "TCP"
  vpc_id      = "${var.vpc_id}"

  stickiness {
      enabled = false
      type = "lb_cookie"
  }

  depends_on = ["aws_lb.heimdall-internal"]
}

resource "aws_lb_target_group" "heimdall-external" {
  name        = "${var.name}-heimdall-external"
  port        = 80
  protocol    = "TCP"
  vpc_id      = "${var.vpc_id}"

  stickiness {
      enabled = false
      type = "lb_cookie"
  }

  depends_on = ["aws_lb.heimdall-external"]
}

resource "aws_lb_listener" "heimdall-internal" {
  load_balancer_arn = "${aws_lb.heimdall-internal.arn}"
  port              = "80"
  protocol          = "TCP"

  default_action {
    type             = "forward"
    target_group_arn = "${aws_lb_target_group.heimdall-internal.arn}"
  }
}

resource "aws_iam_server_certificate" "heimdall" {
  name             = "${var.name}-heimdall"
  certificate_body = "${file("${var.cert}")}"
  private_key      = "${file("${var.private_key_cert}")}"
}

resource "aws_lb_listener" "heimdall-external" {
  load_balancer_arn = "${aws_lb.heimdall-external.arn}"
  port              = "443"
  protocol          = "TLS"
  ssl_policy        = "ELBSecurityPolicy-2016-08"
  certificate_arn   = "${aws_iam_server_certificate.heimdall.arn}"

  default_action {
    type             = "forward"
    target_group_arn = "${aws_lb_target_group.heimdall-external.arn}"
  }
}

resource "aws_ecs_service" "heimdall" {
  name            = "${var.name}_heimdall"
  cluster         = var.cluster_id
  task_definition = aws_ecs_task_definition.heimdall.arn

  load_balancer {
    container_name   = "${var.name}_heimdall"
    container_port   = 8080
    target_group_arn = "${aws_lb_target_group.heimdall-internal.id}"
  }

  load_balancer {
    container_name   = "${var.name}_heimdall"
    container_port   = 80
    target_group_arn = "${aws_lb_target_group.heimdall-external.id}"
  }

  desired_count = 1
}
