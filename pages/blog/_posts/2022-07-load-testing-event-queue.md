---
focus: false
heading: "Load testing an event-driven message queue"
subtitle: How to quickly run load tests on event-driven queues via K6
image: "/assets/blog/k6-load-test.png"
date: 2022-07-29
---

Inngest is an *event-driven queue*.  It differs from typical queues (like SQS, Celery) because it accepts JSON events via HTTP to triggers functions, and it lets you do novel things like data governance with event schemas, proper function versioning, historical event replay, user attribution, or blue green deploys — stuff we’ve come to expect from modern tooling.

You might say the *core* use case of queues and event driven systems is to increase availability and scale.  So… In order to be highly available, we need to make sure the HTTP endpoint that accepts events is as fast as possible.  And we need to know how many events we can handle from a single event API.

## Load testing an event-driven queue

Every time you’re building interconnected systems (eg. HTTP or gRPC APIs) it’s good to know your metrics and limits.  To do that, you’ll be reaching for a load testing or *benchmarking* tool.

One of the classics is `ab` — [ApacheBench](https://www.notion.so/Load-testing-an-event-driven-message-queue-w-k6-555ee98bffd54e0c9159e11339551785), designed to test Apache servers.  Long gone are the days where people reach for Apache and good ol’ GCI-BIN.  Nowadays, most people are using HTTP2/QuiC and, accordingly, there are more modern tools available to benchmark.  Some of the examples:

- [https://github.com/wg/wrk](https://github.com/wg/wrk), one of the early event-loop based systems from 2013.  This can generate *significant* load, and comes with Lua scripting.  It really set the stage for…
- [https://github.com/grafana/k6](https://github.com/grafana/k6), a modern load testing tool written in Go, capable of generating load with complex requests defined in JS-based scripts, with many metric options

The easiest *modern* benchmarking tool to set up and use (in our opinion) is k6.io.  We’ll dive in to a *basic* load testing **test using K6, showing how we used it to test our events API for our event-driven queue.

## About K6

K6 is a Go-based load testing tool which makes performance testing, well, easy.  It allows you to define small JS-based tests, ramp up load, then export the results in many different formats for visualization, CI/CD, etc.

Here’s an example test which submits a small JSON payload to an API:

```go
import http from 'k6/http';

export default function () {
  const data = '{"status":200}';
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };
  http.post('https://www.example.local', data, params);
}
```

The interesting thing here is that a script can do more than one request.  In essence, a script is a user making request — so you can chain together things such as “submit a ticket”, “read a ticket”, etc. 

You can also do more complex things, such as organize and tag requests by group for output, and add *assertions* for the HTTP responses returned from your service.

## Running a test

For Inngest, we accept HTTP events and nothing else, so it’s easy for us to do a simple POST and ramp up virtual users (VUs) as follows:

```go
k6 run --vus 25 --duration 30s ./post.js
```

This is going to ramp up a bunch of users to execute the script continually for 30 seconds, then print a bunch of output such as:

```go
execution: local
     script: ./post.js
     output: -

  scenarios: (100.00%) 1 scenario, 25 max VUs, 1m0s max duration (incl. graceful stop):
           * default: 25 looping VUs for 30s (gracefulStop: 30s)

running (0m30.0s), 0/25 VUs, 209441 complete and 0 interrupted iterations
default ✓ [======================================] 25 VUs  30s

     data_received..................: 20 MB  656 kB/s
     data_sent......................: 169 MB 5.6 MB/s
     http_req_blocked...............: avg=683ns    min=333ns   med=500ns    max=823.15µs p(90)=958ns   p(95)=1.25µs
     http_req_connecting............: avg=2ns      min=0s      med=0s       max=146.12µs p(90)=0s      p(95)=0s
     http_req_duration..............: avg=686.06µs min=68.37µs med=324.87µs max=41.93ms  p(90)=1.23ms  p(95)=2.27ms
       { expected_response:true }...: avg=686.06µs min=68.37µs med=324.87µs max=41.93ms  p(90)=1.23ms  p(95)=2.27ms
     http_req_failed................: 0.00%  ✓ 0          ✗ 209441
     http_req_receiving.............: avg=10.9µs   min=2.33µs  med=8.29µs   max=9.96ms   p(90)=17.54µs p(95)=23.7µs
     http_req_sending...............: avg=6.19µs   min=2.08µs  med=3.66µs   max=10.05ms  p(90)=6.33µs  p(95)=8.58µs
     http_req_tls_handshaking.......: avg=0s       min=0s      med=0s       max=0s       p(90)=0s      p(95)=0s
     http_req_waiting...............: avg=668.96µs min=63.12µs med=310.45µs max=41.9ms   p(90)=1.21ms  p(95)=2.24ms
     http_reqs......................: 209441 6981.16637/s
     iteration_duration.............: avg=712.69µs min=81.58µs med=349.91µs max=41.96ms  p(90)=1.26ms  p(95)=2.31ms
     iterations.....................: 209441 6981.16637/s
     vus............................: 25      min=25        max=25
     vus_max........................: 25      min=25        max=25

k6 run --vus 25 --duration 30s ./post.js  13.30s user 6.14s system 63% cpu 30.576 total
```

By default, this shows you the p90 and p95 latencies for the requests sent — [though this is easy to customize, including streaming output (as events) to different services](https://k6.io/docs/getting-started/results-output/).

Honestly, the [K6 documentation](https://www.notion.so/Reactive-Summit-2022-Proposal-1c03fa8c63d84ea58175ad4cb2282109) is some of the best documentation online we’ve seen.  I’d highly recommend a quick browse through to understand all of the options available.

### Setting up the infra for real-world tests

The above shows a test from your own machine to… your own machine, no internet required.  This isn’t going to give you a good sense of real-world usage, even if it does help with CPU and memory profiling.

In order to really understand how many requests you can receive, you need to set up your services on new infrastructure and test it from an external machine — eg. an EC2 machine which connects to your services via the *public internet*.

You can also use K6’s cloud to handle running the tests, though your services should still be set up on real-world, prod-like infra to understand its limitations.

## Results

We set up our basic [self-hosting stack](https://github.com/inngest/inngest/tree/main/hosting-stacks/aws-managed) using AWS’ managed services.  That is, we hosted our event API on ECS, then pushed events onto SQS and consumed them via another ECS service.

The results we cared about were number of requests/second, the number that failed, and the request duration.  Here are the results on a 1GB ram / 0.5 vCPU instance:

110 requests/second:

```go
http_req_duration..............: min=6.32ms  med=7.71ms  avg=8.84ms  max=80.61ms  p(99)=32.86ms  p(99.9)=74.57ms
     http_req_failed................: 0.00%  ✓ 0
     http_reqs......................: 1113   111.211444/s
```

300 requests/second:

```go
http_req_duration..............: min=6.76ms  med=91.72ms avg=83.18ms max=277.32ms p(99)=187.25ms p(99.9)=199.77ms
     http_req_failed................: 0.00%  ✓ 0
     http_reqs......................: 6009   299.721533/s
```

We can see that latency increases (dramatically) with load, to the point where we recommend 1 vCPU per 150 requests/second.  That is, right now until we optimize our self-hosting infrastructure.

## Conclusion

K6 was able to help us quickly and easily get some initial benchmarks for self-hosted Inngest, which allows users to understand the specs necessary for each of our services.  It’s also a good tool for helping understand sustained load when running eg. pprof in development.

**The cloud**

Alternatively, if you want to build event-driven background functions that scale with zero infra and zero setup, you can always sign up with our cloud.  It lets you push millions of events per month and run functions with zero setup — plus you can use the same open-source services to test your functions and setup locally with no effort.  [Sign up here to get started.](https://www.inngest.com/sign-up?ref=load-test-post)
