terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.22"
    }
  }

  required_version = ">= 1.2.0"
}

provider "aws" {
  # Change your region to... wherever you want :)
  region  = "us-east-2"
}

# Infrastructure
#
# First, we're going to set up the core infrastructure required
# for our services.  We'll take care of deploying the services,
# eg. the event API and runners, after.
# 
# NOTE: This does not set up VPCs, security groups, etc: it only
# shows an example of the infrastructure required and how to host
# and deploy Inngest's services.


# Create our event stream via SQS
resource "aws_sqs_queue" "eventstream" {
  name                      = "eventstream"
  delay_seconds             = 0
  max_message_size          = 262144
  message_retention_seconds = 604800 // Store messages for up to 7 days
  fifo_queue                = false
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.events_deadletter.arn
    maxReceiveCount     = 5
  })

  tags = {
    Environment = "production"
    Category    = "events"
    Service     = "event-stream"
  }
}

# Create our queue for steps via SQS
resource "aws_sqs_queue" "stepqueue" {
  name                      = "stepqueue"
  delay_seconds             = 0
  max_message_size          = 262144
  message_retention_seconds = 1209600 // Store messages for up to 14 days
  fifo_queue                = false

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.stepqueue_deadletter.arn
    maxReceiveCount     = 5
  })

  tags = {
    Environment = "production"
    Category    = "execution"
    Service     = "queue"
  }
}

# Create a dead-letter queue for events
resource "aws_sqs_queue" "events_deadletter" {
  name = "events-deadletter"
}

# Create a dead-letter queue for steps
resource "aws_sqs_queue" "stepqueue_deadletter" {
  name = "stepqueue-deadletter"
}


# Create an Elasticache replication group for hosting state.
resource "aws_elasticache_replication_group" "state" {
  automatic_failover_enabled  = true
  multi_az_enabled            = true
  at_rest_encryption_enabled  = true
  engine                      = "redis"
  engine_version              = "6.x"

  // Create one primary and one replica.
  num_cache_clusters          = 2
  replication_group_id        = "state-rep-group"
  preferred_cache_cluster_azs = ["us-east-2a", "us-east-2b"]
  description                 = "Stores state for each function run"
  node_type                   = "cache.t4g.small" # Change this size for non-test deployments.
  port                        = 6379
}

# Now that we've specified the core infra, let's render out template
# into AWS secrets manager with the creds for each service.
resource "template_file" "config" {
  template = file("${path.module}/config.tpl.cue")
  vars = {
    REDIS_HOST     = aws_elasticache_replication_group.state.primary_endpoint_address
    EVENT_SQS_URL  = aws_sqs_queue.eventstream.url
    QUEUE_SQS_URL  = aws_sqs_queue.stepqueue.url
  }
}
# Ensure the secret container is defined.
resource "aws_secretsmanager_secret" "config" {
  name = "config"
}
# Write this version of the config.
resource "aws_secretsmanager_secret_version" "example" {
  secret_id     = aws_secretsmanager_secret.config.id
  secret_string = template_file.config.rendered
}


# Deploy the ECS containers
#
# Create a cluster for each ECS task.  This will be hosted on Fargate
# for ease of use.
resource "aws_ecs_cluster" "services" {
  name = "inngest-services"
  setting {
    name  = "containerInsights"
    value = "enabled"
  }
}


# We need to create a new IAM role which allows the task definitions
# to read our secrets.
resource "aws_iam_policy" "config" {
  name        = "inngest-config-policy"
  path        = "/"
  description = "Allows access to Inngest config and queues"
  # Terraform's "jsonencode" function converts a
  # Terraform expression result to valid JSON syntax.
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      # Allow logging
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogStream",
          "logs:CreateLogGroup",
          "logs:PutLogEvents",
        ],
        Resource = "*"
      },
      # Allow secrets access
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetResourcePolicy",
          "secretsmanager:GetSecretValue",
          "secretsmanager:DescribeSecret",
          "secretsmanager:ListSecretVersionIds"
        ],
        Resource = [
          # Only allow access to our config secret.
          aws_secretsmanager_secret.config.arn
        ]
      },
      # Allow publish/subscribe to our queues
      {
        Effect = "Allow"
        Action = [
          "sqs:ChangeMessageVisibility",
          "sqs:ChangeMessageVisibilityBatch",
          "sqs:DeleteMessage",
          "sqs:DeleteMessageBatch",
          "sqs:GetQueueAttributes",
          "sqs:GetQueueUrl",
          "sqs:ListDeadLetterSourceQueues",
          "sqs:ReceiveMessage",
          "sqs:SendMessage",
          "sqs:SendMessageBatch",
        ],
        Resource = [
          # Only allow access to our config secret.
          aws_sqs_queue.eventstream.arn,
          aws_sqs_queue.stepqueue.arn,
          aws_sqs_queue.events_deadletter.arn,
          aws_sqs_queue.stepqueue_deadletter.arn,
        ]
      },
    ]
  })
}
resource "aws_iam_role" "config_role" {
  name = "config_role"
  # Allow EC2 and ECS tasks to use this role.
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Sid    = ""
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      },
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Sid    = ""
        Principal = {
          Service = "ecs-tasks.amazonaws.com"
        }
      },
    ]
  })
}
# Let the above role use our previously defined policy.
resource "aws_iam_role_policy_attachment" "config_role_attach" {
  role       = aws_iam_role.config_role.name
  policy_arn = aws_iam_policy.config.arn
}


# Create new task definitions for the event API and runner.
# 
# It's hard to mount files to fargate-based ECS containers, so we use
# the CONFIG env var to host our config file's contents.
#
# You can also store this config file in AWS secrets manager or AWS
# config manager;  these are more suitable.
resource "aws_cloudwatch_log_group" "eventapi" {
  name = "/ecs/ecs-eventapi-task"
}
resource "aws_ecs_task_definition" "eventapi" {
  family                   = "eventapi"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc" # Required for fargate.
  task_role_arn            = aws_iam_role.config_role.arn
  execution_role_arn       = aws_iam_role.config_role.arn
  cpu    = 512
  memory = 1024
  container_definitions = jsonencode([
    {
      name      = "eventapi"
      image     = "inngest/inngest:latest"
      cpu       = 512
      memory    = 1024
      essential = true
      command   = ["inngest", "serve", "events-api"]
      portMappings = [
        {
          containerPort = 80
          hostPort      = 80
        }
      ]
      environment = [
        {"name": "ENV", "value": "test"}
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.eventapi.name
          awslogs-stream-prefix = "ecs"
          awslogs-region        = "us-east-2"
        }
      }
      secrets = [
        # Add our config as a secret, as it may contain secret values
        # such as database passwords.
        { "name": "INNGEST_CONFIG", valueFrom: aws_secretsmanager_secret.config.arn }
      ]
      // TODO: Healthchecks, see
      // https://github.com/inngest/inngest/issues/162
    },
  ])
}
resource "aws_ecs_service" "eventapi" {
  name            = "eventapi"
  cluster         = aws_ecs_cluster.services.id
  task_definition = aws_ecs_task_definition.eventapi.arn
  desired_count   = 2
  launch_type     = "FARGATE"
  network_configuration {
    # NOTE: You probably want to configure a NAT or VPC endpoints instead
    # of public IPs.
    assign_public_ip = true
    # NOTE: This uses the default subnets.  See the bottom of the file
    #       where this is defined;  you likely want to use your own VPC.
    subnets = [
      aws_default_subnet.default_az1.id,
      aws_default_subnet.default_az2.id,
      aws_default_subnet.default_az3.id
    ]
  }
  load_balancer {
    target_group_arn = aws_alb_target_group.eventapi.arn
    container_name   = "eventapi"
    container_port   = 80
  }
}
# Create a load balancer for the API
resource "aws_lb" "eventapi" {
  name               = "eventapi-alb"
  internal           = false
  load_balancer_type = "application"
  subnets = [
    aws_default_subnet.default_az1.id,
    aws_default_subnet.default_az2.id,
    aws_default_subnet.default_az3.id
  ]
}
resource "aws_alb_target_group" "eventapi" {
  name        = "eventapi-alb-alb-tg"
  target_type = "ip"
  port        = 80
  protocol    = "HTTP"
  vpc_id      = aws_default_vpc.default.id
}
resource "aws_alb_listener" "eventapi_http" {
  load_balancer_arn = aws_lb.eventapi.id
  port              = 80
  protocol          = "HTTP"
  # Typically, you would redirect this with HTTP
  # once you set up an SSL cert:
  #
  # default_action {
  #   type = "redirect"
  #
  #   redirect {
  #     port        = 443
  #     protocol    = "HTTPS"
  #     status_code = "HTTP_301"
  #   }
  # }
  default_action {
    target_group_arn = aws_alb_target_group.eventapi.id
    type             = "forward"
  }
}


# Add the runner as an ECS task
resource "aws_cloudwatch_log_group" "runner" {
  name = "/ecs/ecs-runner-task"
}
resource "aws_ecs_task_definition" "runner" {
  family                   = "runner"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc" # Required for fargate.
  task_role_arn            = aws_iam_role.config_role.arn
  execution_role_arn       = aws_iam_role.config_role.arn
  cpu    = 512
  memory = 1024
  container_definitions = jsonencode([
    {
      name      = "runner"
      image     = "inngest/inngest:latest"
      cpu       = 512
      memory    = 1024
      essential = true
      command   = ["inngest", "serve", "runner"]
      portMappings = [
        {
          containerPort = 80
          hostPort      = 80
        }
      ]
      environment = [
        {"name": "ENV", "value": "test"}
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.runner.name
          awslogs-stream-prefix = "ecs"
          awslogs-region        = "us-east-2"
        }
      }
      secrets = [
        # Add our config as a secret, as it may contain secret values
        # such as database passwords.
        { "name": "INNGEST_CONFIG", valueFrom: aws_secretsmanager_secret.config.arn }
      ]
      // TODO: Healthchecks, see
      // https://github.com/inngest/inngest/issues/162
    },
  ])
}
resource "aws_ecs_service" "runner" {
  name            = "runner"
  cluster         = aws_ecs_cluster.services.id
  task_definition = aws_ecs_task_definition.runner.arn
  desired_count   = 2
  launch_type     = "FARGATE"
  network_configuration {
    # NOTE: You probably want to configure a NAT or VPC endpoints instead
    # of public IPs.
    assign_public_ip = true
    # NOTE: This uses the default subnets.  See the bottom of the file
    #       where this is defined;  you likely want to use your own VPC.
    subnets = [
      aws_default_subnet.default_az1.id,
      aws_default_subnet.default_az2.id,
      aws_default_subnet.default_az3.id
    ]
  }
}

# EC2 executor creation
module "asg" {
  source = "terraform-aws-modules/autoscaling/aws"
  name   = "executors"

  # Change this to a key pair name you own.
  key_name = "EXAMPLE-KEY"

  # Launch configuration
  user_data       = base64encode(<<-EOT
  #!/bin/bash
  sudo yum install -y docker
  usermod -a -G docker ec2-user
  chkconfig docker on
  systemctl enable docker.service
  service docker start

  docker run -d --restart always \
    --name executor \
    --env INNGEST_CONFIG="$(aws secretsmanager get-secret-value --region us-east-2 --secret-id ${aws_secretsmanager_secret.config.arn })" \
    -v /var/run/docker.sock:/var/run/docker.sock \
    inngest/inngest:latest \
    inngest serve executor
  EOT
)
  # NOTE: This runs Amazon Linux 2 on Linux 5.10.
  image_id        = "ami-02d1e544b84bf7502"
  instance_type   = "t3a.small"
  security_groups = [aws_security_group.executor_example_sg.id]
  block_device_mappings = [
    {
      device_name = "/dev/xvda"
      no_device   = 0
      ebs = {
        delete_on_termination = true
        encrypted             = true
        volume_size           = 50
        volume_type           = "gp2"
      }
    }
  ]

  create_iam_instance_profile = true
  iam_role_name               = "executors-role"
  iam_role_path               = "/ec2/"
  iam_role_description        = "Executor role for SQS, logs, and secrets"
  iam_role_policies = {
    config = aws_iam_policy.config.arn
  }

  # Auto scaling group
  vpc_zone_identifier       = [
    aws_default_subnet.default_az1.id,
    aws_default_subnet.default_az2.id,
    aws_default_subnet.default_az3.id
  ]
  health_check_type         = "EC2"
  min_size                  = 1
  max_size                  = 2
  desired_capacity          = 1
  wait_for_capacity_timeout = 0

  scaling_policies = {
    my-policy = {
      policy_type               = "TargetTrackingScaling"
      target_tracking_configuration = {
        predefined_metric_specification = {
          predefined_metric_type = "ASGAverageCPUUtilization"
        }
        target_value = 50.0
      }
    }
  }
}
# Allow any egress
resource "aws_security_group" "executor_example_sg" {
  egress {
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }
}


# !!! NOTE: We have not isolated this configuration using its own VPC.
#           Instead, we're using the default VPC and subnets that come
#           with your own AWS account, as an example.
resource "aws_default_vpc" "default" {}
resource "aws_default_subnet" "default_az1" { availability_zone = "us-east-2a" }
resource "aws_default_subnet" "default_az2" { availability_zone = "us-east-2b" }
resource "aws_default_subnet" "default_az3" { availability_zone = "us-east-2c" }
