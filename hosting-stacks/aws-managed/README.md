# AWS Managed Stack

This is an **example** stack hosted on AWS which uses the following services:

| Inngest service  | AWS component | Why | Potential other choices |
| ---              | ---           | --- | --- |
| **Event API**    | ECS           | A simple managed service | EKS, EC2 |
| **Runner**       | ECS           | A simple managed service | EKS, EC2 |
| **Executors**    | EC2 via an ASG | Runs steps as docker containers within the EC2 machines | ECS, EKS (see notes) |
| **Event stream** | SQS           | Handles inbound events without managing capacity | Kinesis, Elasticache |
| **Queue**        | SQS           | A managed queue with (limited) delayed messaging | - |
| **State**        | Elasticache   | Managed Redis, a compatible state store | - |

<br />

## Getting started

1. Browse the example Terraform to view the infrastructure needed
2. Read the inngest.cue configuration to inspect and see the service's config
   for AWS.
3. Amend the terraform to include:
  - Your own security groups
  - Your own VPC definitions (this example uses the default VPC)
  - Your own keypairs
  - Your own logging & monitoring

### Assumptions

We assume that the following drivers will be used to run steps:

- Docker, using the local Docker client on each EC2 machine
- HTTP

### Notes

This is as 'serverless' as possible, with the only state being the executors
running on EC2 via an auto-scaling group.  We assume this is needed to run
docker-based steps via the Docker client local to the EC2 machine.

It's possible to switch out the basic Docker client for a Nomad or Kubernetes
driver, which removes the need for the EC2 instance - as Docker-based steps will
be launched on your existing container infrastructure.

**Unsupported features**

Right now this doesn't host the core API.  In a future version (end of July, 22)
we will update this stack to incorporate a basic RDS/Aurora machine to store
function & action state.

This also doesn't provide VPC or NAT configuration for machines.
