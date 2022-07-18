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

### Benchmarks

A single ECS instance with 1GB ram and 512 CPU (0.5/vCPU) has the following event
ingestion benchmarks:

**110 requests/second:**

```
     http_req_duration..............: min=6.32ms  med=7.71ms  avg=8.84ms  max=80.61ms  p(99)=32.86ms  p(99.9)=74.57ms
     http_req_failed................: 0.00%  ✓ 0
     http_reqs......................: 1113   111.211444/s
```

**300 requests/second:**

```
     http_req_duration..............: min=6.76ms  med=91.72ms avg=83.18ms max=277.32ms p(99)=187.25ms p(99.9)=199.77ms
     http_req_failed................: 0.00%  ✓ 0
     http_reqs......................: 6009   299.721533/s
```

Latency increases with CPU load;  it's possible to handle 100 requests/second with a p99 latency of 30ms,
or 300 requests/second with a p99 latency of 187ms.

We recommend 1GB ram / 0.5 vCPU for each ~100 requests/second within the events API.

The runner can handle ~100 requests/second with 1GB ram / 0.5 vCPU under light load (ie. < 5% of events
which trigger functions).  We recommend having at least 2x the number of runner services than you do
event API services;  runner services have higher CPU load executing expressions and querying functions
to run.
