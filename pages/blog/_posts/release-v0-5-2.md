---
heading: "Inngest: OS v0.5.2 released"
subtitle: Our next release improving rollbacks and developer UX
date: 2022-08-09
image: "/assets/blog/release-v0.5.0.jpg"
tags: release-notes
---
[Inngest v0.5.2 is here](https://github.com/inngest/inngest/releases)!  This patch introduces a few new pieces of functionality, as well as various fixes and improvements.  The key pieces are:

 

- **Strict function rollbacks**, meaning any time you change a single step’s code you can roll back to previous versions.
- **Improved `inngest init`,** showing you every event you’ve seen when creating functions, plus the ability to bypass questions via flags.

Read more about our [future plans in our roadmap](https://github.com/orgs/inngest/projects/1), and if you want to propose new features or ideas feel free to [start a discussion](https://github.com/inngest/inngest/discussions) or [chat with us in discord](https://www.inngest.com/discord). Let’s dive in!

## Strict rollbacks

Inngest automatically versions every step function that you deploy.  This allows you to see a full change log for each of your functions, plus the ability to immediately roll back to a previous version if desired.

We’ve updated our function configuration format to make rollbacks *stricter —* to always ensure that you roll back every step to the correct state for each version.

## Inngest init

Using `inngest init` is the easiest way to create new serverless functions.  You’ll often want to create a function which responds to an event that you’ve already processed within your account.  In the latest version, `inngest init` automatically fetches every event you’ve seen in your account so that you can create new functions that run any time these events are seen again in the future.

![v0.5.2](/assets/blog/init-0.5.2.gif)

We’re using the new init flags and process in our guide to running [Prisma.js background jobs](https://www.inngest.com/docs/guides/prisma-background-jobs).

Happy building!  If you have any questions or feedback feel free to [join us in our discord](https://www.inngest.com/discord) — we’re hacking there!
