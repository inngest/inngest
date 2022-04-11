<h1 align="center">
        <img src="https://www.inngest.com/logo-white.svg" alt="Logo" width="140" height="70"><br />
        A CLI for event-driven serverless functions
</h1>

<br />


`inngestctl` allows you to locally build and test **event-driven** serverless functions.  Using this CLI, you can:

- **Scaffold new serverless functions in seconds**
- **Locally run functions** for fast development
- **Test** functions **with live, historical data**
- Deploy functions instantly
- Time-travel and deploy versions as if they existed in the past 🕗🔙🕘

Using events for serverless functions makes development **fast, easy, safe, and fun!**.  Our aim is to make event-driven development _easier than ever_.

<br />
<br />

- <a href="#installation">Installation</a>
- <a href="#getting-started-45-second-guide">Getting started & developing functions</a>
- <a href="#how-it-works">How it works</a>

<br />
<hr />
<br />

<h3 align="center">Installation</h3>
<br />

Quick start:

```bash
# downloads inngestctl for your os+arch into ./inngestctl
curl -sfL https://raw.githubusercontent.com/inngest/inngestctl/main/install.sh | sh && \
  sudo mv ./inngestctl /usr/local/bin/inngestctl
```

<b>Via the install script</b>


1. Run `curl -sfL https://raw.githubusercontent.com/inngest/inngestctl/main/install.sh | sh` to automatically download the latest binary for your OS and arch
2. Move the new `./inngestctl` file into your `$PATH` (eg. `mv ./inngestctl /usr/local/bin`)

<b>Manually</b>

1. Download a [pre-compiled binary](https://github.com/inngest/inngestctl/releases) and place the binary in your path
2. Move the new `./inngestctl` file into your `$PATH` (eg. `mv ./inngestctl /usr/local/bin`)

<br />
<hr />
<br />

<h3 align="center">Getting started: 45 second guide</h3>
<br />

It's *really easy*:

1. Run `inngestctl init` to scaffold a new function.  We'll ask you for your *function name*; the *event that triggers the function*; and *the function language*.
2. `cd` into your new function
3. Run `inngestctl run` to *run your function locally* using data generated from the event definition
4. Run `inngestctl deploy` to deploy your function.

<br />
<hr />
<br />

<h3 align="center">How it works</h3>
<br />

Inngest changes the way serverless functions are developed, deployed, and triggered.  We believe that event-driven development is a great way to build and architect flexible, maintainable, decoupled systems — but it's impossibly hard to do right.

We provide everything you need for killer, world class event-driven systems out of the box.  Here's an overview of how we work:

1. **We provide a central API for publishing events**.  You can send events with a simple HTTP request ([here are the docs](https://www.inngest.com/docs/event-http-api-and-libraries)).
2. **We record all events received in your system**.  This can be from weeks to _years_.
3. **Events trigger serverless functions automatically**.  We automatically trigger serverless functions with events as the request data.

#### Why events?

Using events instead of raw requests allows us to:

1. **Fully-type your event payloads**, with schema versions as each event changes over time
2. **Enforce event schemas if desired**, preventing data issues from causing bugs
4. **Generate an audit trail** of what happened in your system, when, and who was responsible
5. **Automatically retry functions on failure**, improving reliability
6. **Run functions with historic events**, allowing you to deploy functions with old data as if they were deployed in the past
7. **Handle complex multi-step flows**, eg. run a step-function on add to cart; wait for the purchase event;  then run a step on purchase _or a step on timeout_.  This is killer.

Nothing's stopping you from developing regular ol' HTTP based serverless functions using our tooling, either.  You can still build functions intended for AWS Lambda & API gateway using this CLI, getting local testing for free.

<br />
<hr />
<br />

<b>Telemetry</b>

Telemtry is currently **extremely limited**.  First, some commitments:

- We never track personal information (eg. IP) from the CLI
- We only ever want to record _metrics_ for product improvement
- For example, we want to answer "Is generating test data for XYZ language heavily used?"
 
We're a small team and want to make sure we're building the right things.  You can opt out by exporting `DO_NOT_TRACK=1` before running `inngestctl`;  we will never send requests with this env variable set.

### License

This product is released under the GPL-3 license.  It's copyleft.  We want to make sure our work is open-source and it's always appreciated when improvements are committed back to the community.
