---
focus: true
heading: "Building a real-time websocket app using SvelteKit"
subtitle: Our experience building https://typedwebhook.tools in 2 days using SvelteKit.
date: 2022-02-18
order: 5
---

<div className="blog--callout">

We're Inngest.  We make it easy to run serverless functions in response to events.  How?  We're an event mesh that fully types, records, and analyzes all of your events, then runs functions instantly we receive them.  It makes building reliable serverless functions much easier.  [You can get started for free](/sign-up).
</div>

We recently built a [webhook testing tool which auto-generates types for each request](https://typedwebhook.tools).  It's called https://typedwebhook.tools, it's free, and it's meant to make development easier.

While there are things you can use to see webhook payloads, most of the time you want to work with that request body — and for that, you need to generate types.  In this post we’ll walk through how we built the real-time UI for the webhook tool using Svelte.  Here’s what we’ll discuss:

1. What is Svelte?
2. Why did we choose Svelte (over React)?
    1. Scope of the project
    2. Svelte vs SvelteKit
3. Impressions
4. Comparison to React
5. Conclusion

### What is Svelte?

Svelte is a frontend framework for building *progressively enhanced* web apps.  It has a few key ideas:

- **No runtime**.  It generates components at build time (using Vite), and doesn’t ship a runtime.  In short, this means faster and *much* smaller client code.
- **Common-denominator tooling.**  It uses JS (or TS), CSS, and HTML.  Tools that you already know.
- **All-in-one**.  It comes with state management, transitions, animations, etc. all built in. State management is effortless and reactive.

In short, it’s made for performant webapps, developer efficiency, and developer UX (which is one of *our* core tenets).

### Why did we choose SvelteKit (vs React)?

We’re *really* familiar with React.  We’ve all shipped many apps using it - whether it’s web apps or React Native.  So why take the productivity loss of using a new framework for building a UI?

Well, we like learning, but that’s not all. Svelte has a few features that made it a good fit for the project.  As we mentioned, we were making a tool that allows you to create free, unique URLs you can use as webhook endpoints for testing.  We ingest the webhook requests, store the headers & body for 60 minutes, then show you the data.

The novel part: we also auto-generate typescript types, cue types, and JSON schemas for every JSON payload we see so that you can work with the data easily ✨.  Cue is amazing.  We’ll be writing about that later.

So here's why we chose SvelteKit:

- Using Vite, it builds *fast* in dev.  And it has HMR with state persistence out of the box.  Somehow this is *consistently* broken in every react app, no matter what kit you use.
- It has SSR out of the box.  It’s built for progressive enhancement, and [configuring pre-rendering is the easiest I’ve seen](https://kit.svelte.dev/docs/page-options#prerender).  Next has this — we use next for our site and built our own Next.JS documentation plugin for our docs — but dealing with state in NextJS is a little harder.  We’ve found Next is perfect for static sites, or sites with minor interactivity.  Which brings us to...
- State management is *easy.*  [It’s easy to work with stores](https://svelte.dev/tutorial/writable-stores).  You can (broadly speaking) use the stores from anywhere:  no top-level context necessary.  It’s out of the box, and it works everywhere.  It’s also reactive, and it’s easy to integrate with things that don’t live in components (ahem, websockets).

And, because SvelteKit comes with a standard way of doing things (CSS, JS, forms, state, routing), it’s easy to work with and it’s easy to share amongst devs.  This is why we chose SvelteKit over pure Svelte.  It’s easy to get set up and running with your entire framework — think a mixture of NextJS and reate-react-app for Svelte.

### Impressions

Here’s a distilled quick take on Svelte(Kit):

1. It’s easy to get started and dive in. You get SvelteKit up and running by using `npm init svelte@next`.  It scaffolds a project *just like* CRA does.  You can scaffold either a blank project or the demo project.  If you learn best by hacking around then scaffolding an existing project to see how it works is *great*.
2. Routing is plain and simple, just like Next.  It uses the same principles:  folder paths representing routes, `[params]` for dynamic content, etc.
3. It takes a little minute to get used to their way of defining components (a script tag, some HTML, and some CSS).  It's unique but quite nice.  It enforces one component per file, makes CSS easy, and defining arbitrary JS for components is easy.
4. The standard `$lib/Foo.svelte` imports break Storybook and makes type checking more difficult.  It also breaks neovim LSP’s jump to definitions, etc.  With some config changes this can be fixed, but out of the box it’s a small hurdle.
5. State management is *so easy*.  Literally create a store and you’re off to the races.  It’s easy to mutate the store.  It took about 3 minutes for us to integrate websocket mutations into the store and have this propagate across the app.
6. Viewing state feels weird initially (but works well).  Let’s say we have a store in a variable called `pages`.  You access the store’s subscribe & mutate functionality using `pages`.  And you access the data with a `$` prefixed version: `$pages`.  [There’s some magic going on here](https://svelte.dev/docs#component-format-script-4-prefix-stores-with-$-to-access-their-values).  It’s also [explained well here](https://svelte.dev/tutorial/auto-subscriptions).  It’s kind of easy to forget on day one but you get used to it — a minor gotcha.
7. Derived stores, store bindings — it’s well thought out and concise.  It makes me really happy that we’ve progressed from the verbosity of the past.
8. [Reactive statements using labels are *kind of* funky.](https://svelte.dev/docs#component-format-script-3-$-marks-a-statement-as-reactive)  If you come from React, you might expect that declaring variables “just works”.  It’s a little bit easier to render content that *doesn’t* update because you forgot to mark things as reactive.  It’s easy enough to mark every rendered variable in a template as reactive.  Wondering if this could be a default and what the implications are 🤔
9. The “one” way of doing CSS is great for packages. Most every package uses CSS variables to manage themes.  It’s easy to adjust with wrappers ([here’s an example](https://github.com/zerodevx/svelte-toast#theming)).  It pushes actual standards.
10. That said, it’s easy to miss [emotion.sh](https://emotion.sh/docs/introduction) with the good ol’ nesting of selectors, the trusty `&`, and the fancy pre-processing.
11. The docs are fragmented between [examples](https://svelte.dev/tutorial/basics) and [classic documentation](https://svelte.dev/docs).  It’s okay, though they could be integrated together inline.
13. This was a *dead* simple app.  Maybe you’d miss redux-style actions on something more complex (not that we use Redux any more, `createReducer` is “good enough”), but a big fat warning here is that we haven’t used Svelte enough to know how it works with a horribly complex UI.
14. Surprisingly, there were Svelte packages for two things we needed:  [a JSON view](https://github.com/zerodevx/svelte-json-view) and a [toast display](https://github.com/zerodevx/svelte-toast).  By the same person!
15. It really is amazingly fast and performant in all aspects — to load, to paint, and to use.  Without any optimization here’s the first lighthouse profile:

<img src="/assets/perf.png" />

### An example: writing realtime websockets

Here's an example of how easy it is to create a reactive store that is updated on each websocket message:

```typescript
import { writable } from 'svelte/store'

type State = {
  requests: Array<Request>;
}

// Create a new store with the given data.
export const state = writable<State>({
  requests: [],
});

export const connect = () => {
  // Create a new websocket
 const ws = new WebSocket("ws://example.com");

  ws.addEventListener("message", (message: any) => {
    // Parse the incoming message here
    const data: Request = JSON.parse(message.data)
    // Update the state.  That's literally it.  This can happen from anywhere:
    // we're not in a component, and there's no nested context.
    state.update(state => ({ ...state, requests: [data].concat(state.requests) }));
  })
}
```

There's more to websockets than this, but in essence state management is about 4 lines of work, with zero context or component relationships to manage.

### Comparisons to react

We like them both.  Honestly, TL;DR: Svelte is fantastic to use, especially for a small project like this.  React is also great, trusted, and works well for complex state.

Svelte’s reactive state management is great, and it’s built in.  Because of React’s proliferation — and in a way, it’s minimalism — there are a lot of competing state layers.  It’s a little harder to get React set up into the same state (🥁🐍), and each React project has different standards to jump into.  Things like “where are shared components, where are top-level router pages, do we use nested routes?” are answered for you in Svelte but not React.

That said, React has an answer for everything.  We lucked out with the two Svelte packages that were used.  React’s ecosystem is *huge*, and that’s a *good thing* (depending on where you sit on package proliferation, NPM, and a whole can of worms we don’t want to go into).

### Conclusion

I’d like to learn more and dabble with Svelte in more complex projects.  It was great to use, super productive, fast to build with, and fast to deploy.  Overall it’s a definite recommend!  That said, React is such a common tool that we feel safer using it for larger projects.  Maybe this will change over time or more than 2 days of Svelte usage :)

Finally, you’ll probably note that we didn’t even dive into type generation here.  We’re planning a to write about how we do that soon.  It involves [Cue](https://cuelang.org), a package we wrote to introspect Cue types from JSON, and a package we wrote to translate Cue to Typescript (including the *good* kind of TS enums).  Cue is our lingua franca of types — for a good reason we’ll go into soon :)
