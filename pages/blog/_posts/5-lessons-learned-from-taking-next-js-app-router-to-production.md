---
focus: true
heading: "5 Lessons Learned From Taking Next.js App Router to Production"
subtitle: "What did we learn from building and shipping our new app with the Next.js 13 App Router?"
image: "/assets/blog/5-lessons-learned-from-taking-next-js-app-router-to-production/featured-image.png"
date: 2023-05-05
author: Igor Gassmann
---

Next.js 13 introduced the new [App Router](https://nextjs.org/docs/app) that offers several new features, including [Nested Layouts](https://nextjs.org/docs/app/building-your-application/routing/pages-and-layouts#nesting-layouts), [Server Components](https://nextjs.org/docs/getting-started/react-essentials#server-components), and [Streaming](https://nextjs.org/docs/app/building-your-application/routing/loading-ui-and-streaming#what-is-streaming). It’s the first open-source implementation that allows developers to fully leverage the primitives brought by [React 18](https://react.dev/blog/2022/03/29/react-v18).

It was in beta for a while, but since [Next.js 13.4](https://nextjs.org/blog/next-13-4), which was just released, it’s now considered production-ready.

We recently redesigned the Inngest Dashboard to account for new features (branch deploys, metrics, logs, etc.). We rewrote it using Next.js 13 and the App Router. If you're thinking about adopting the App Router in your project, here are some of the things we learned to hopefully make that smoother for you.

## Why use the App Router?

Before we dive in, it's good to understand _why_ to use the App Router. [There are several](https://www.youtube.com/watch?v=DrxiNfbr63s) [resources](https://www.youtube.com/watch?v=gSSsZReIFRk) from the Next.js team explaining the benefits of the the new router, but why did _our team_ decide to use it now?

Our previous app was built using [Create React App](https://create-react-app.dev/) and React Router, so switching to Next.js in general gave us some clear benefits:

- Immediate render of the application using [static rendering](https://nextjs.org/docs/app/building-your-application/rendering#static-and-dynamic-rendering-on-the-server) to avoid the blank loading state of a purely client-side SPA.
- [Middleware](https://nextjs.org/docs/app/building-your-application/routing/middleware) to handle auth redirects without an SPA "page flash."
- A file convention-based framework that allows new engineers to ramp up on the project quickly.
- Bundling and code-splitting out-of-the-box for a better end user experience.
- Extremely easy to adopt Edge functions if we wanted to.

Additionally, choosing to use the **App Router** in our new project added the following benefits:

- Thanks to Nested Layouts, we could more easily share UI between routes while preserving state and avoiding expensive re-renders.
- We could use [React Server Components](https://beta.nextjs.org/docs/rendering/server-and-client-components#server-components) to better leverage the server.
- We could display parts of a page sooner, without waiting for all the data to load, thanks to [Streaming](https://beta.nextjs.org/docs/data-fetching/streaming-and-suspense).
- We’d be ready for the future; no need to migrate when the `pages/` directory eventually.

We made the decision that the App Router was the right call starting a new Next.js project. After shipping our app, we believe this was the right decision, but we did learn some things a long the way!

## 1. Understand The Two Caches

The App Router introduces two new caches:

- A [**client-side cache**](https://nextjs.org/docs/app/building-your-application/routing/linking-and-navigating#client-side-caching-of-rendered-server-components) that stores the payload of previously navigated or prefetched routes to help make navigation feel near-instant.
- A [**server-side cache**](https://nextjs.org/docs/app/building-your-application/data-fetching#caching-data) that improves the performance of server-side data fetching

Understanding these caches is key including which cache invalidation rules they follow and how they interact with each other. Since we adopted the App Router when it was still considered experimental, we sometimes found answers in Github (open source FTW!) for questions like how long the client-side cache is preserved (_it's [30 seconds](https://github.com/vercel/next.js/pull/48383) if you're curious!_)

If you aren’t making `GET` requests with the `fetch()` function when fetching on the server, the returned data will be considered static by Next.js unless you remember to use the [`cache()` function](https://nextjs.org/docs/app/building-your-application/data-fetching/caching#per-request-caching). This means the returned data will be fetched at build time, cached, and reused on each request. This is important to understand so your application doesn't end up with stale date when dealing with actual dynamic data.

If you’re building an app that deals largely with dynamic data, one option is to temporarily disable the server-side cache for your route segments by setting `export const dynamic = 'force-dynamic'` in your `layout.tsx` and `page.tsx` files until you feel comfortable with the Next.js cache. This can help you incrementally learn the server-side cache and adopt it correctly.

## 2. Streaming Improved Page Loading Experience

The Streaming feature of the App Router allowed us to display parts of a page sooner without waiting for all the data to load.

For example, we were able to stream the initial content of a page, such as the header and navigation menu, before all the data had finished loading. This allowed users to start interacting with the page sooner, even if some content was still loading, leading to an improved user experience.

<video controls src="/assets/blog/5-lessons-learned-from-taking-next-js-app-router-to-production/streaming-in-action.mp4" />

## 3. URL Search Parameters in a Layout Server Component

It's important to know that [unlike page components](https://nextjs.org/docs/app/api-reference/file-conventions/page#searchparams-optional), the App Router doesn’t make URL search parameters (`?key1=value1&key2=value2`) available to layout server components. This is because a [layout component](https://nextjs.org/docs/app/api-reference/file-conventions/layout) is not re-rendered when the user navigates to a different page within that same layout. The search parameters could change between navigations, leading to the layout component having outdated values for the search parameters. The router works that way to provide faster navigation.

We wanted to implement an optional global filter for our app that would persist in the URL and we originally wanted to include some data from this filter in a layout component. We needed to allow users to select an environment that filters all displayed data (functions, events, and deploys) to that environment.

![The environment selector in the Inngest Dashboard](/assets/blog/5-lessons-learned-from-taking-next-js-app-router-to-production/environment-selector.png)

Our first idea was to add an `env` search parameter to the URL, like this: `https://app.inngest.com/functions?env=staging`. During testing, we learned that since layouts are not re-rendered based on search parameters, it would lead to stale data in our UI. We either needed to remove this data from our layouts _or_ find another solution.

To solve this issue, we converted the search parameter into an route parameter: `https://app.inngest.com/env/staging/functions`. Layout components can receive [dynamic route parameters](https://nextjs.org/docs/app/api-reference/file-conventions/layout#params-optional), which resolved our problems. Since the App Router primarily uses paths for routing, we found that it works best when putting parameters like this into the URL path.

Alternatively, you might be able to use middleware and parallel routes with search params depending on your use case: [check out this thread from Dan Abramov](https://twitter.com/dan_abramov/status/1655269078741786629?s=20).

## 4. The Opinionated File Structure Brings Many Benefits

The App Router is configured by creating [special files](https://nextjs.org/docs/app/building-your-application/routing#file-conventions) within a folder structure. You can use those special files to declare [Suspense](https://react.dev/reference/react/Suspense) and [Error](https://react.dev/reference/react/Component#catching-rendering-errors-with-an-error-boundary) boundaries at multiple nesting levels.

![A section of the Inngest Dashboard’s file structure](/assets/blog/5-lessons-learned-from-taking-next-js-app-router-to-production/file-structure.png)

We can understand our app’s structure and see at which levels the suspense and error boundaries are declared just by looking at our file structure. Before the App Router, we would have to look inside components to find these boundaries. This also has the benefit of promoting the use of those React primitives, which are often neglected by React developers.

Additionally, we can now [colocate](https://nextjs.org/docs/app/building-your-application/routing#colocation) our files with our routes, such as components, tests, and styles. This is especially useful for files that are only used by one route.

## 5. Learning New Technologies & Limited Resources

Adopting a new technology, especially before something is considered _stable_, is always a challenge for a team. With the App Router, it opened up many technologies that we could adopt: new routing, React Server Components, and new caches. For long-time React developers, React Server Components might require you to update your existing mental models for how to structure components. A benefit with App Router is that you can choose whether to use a server or client component. Since we chose to adopt all of this new tech at once, we probably slowed our development process a bit, but that was our own choice and you can choose what to adopt and when.

The official Next.js team did excellent work with their [docs](https://nextjs.org/docs), and it is immensely helpful to learn the basics. Since the App Router is so new, there may not be as many resources such as blog posts, Stack Overflow questions or similar to help you out. If you get stuck, we recommend checking out [GitHub issues](https://github.com/vercel/next.js/issues) and Twitter conversations.

As with any new technology, we're still learning! We still have a couple of things that we're planning to figure into:

- How to properly dedupe fields across multiple GraphQL queries when using React Server Components
- How do you add pagination to a list in a layout server component

With time and more adoption, we'll see more learning resources and best practices emerging from the community, such as this blog post. Before then, it’s important to be patient and persistent in seeking solutions. It’s also helpful to share our experiences and solutions with others to help build a more substantial knowledge base.

When it comes to React Server Components, if you get stuck, we recommend falling back to a client component until you can sort it out. In Next.js, client components still benefit from being [pre-rendered](https://nextjs.org/docs/app/building-your-application/rendering#static-and-dynamic-rendering-on-the-server) on the server, like in the [Pages Directory](https://nextjs.org/docs/pages/building-your-application/rendering#pre-rendering) so it's not a bad temporary trade off.

## Conclusion

The Next.js App Router can provide a lot of benefits that enhance both end user and developer experience. If you choose to adopt it, you should consider the same aspects as you would with any relatively new technology - you need to be patient and sometimes dig a little deeper.

We started building our app when the App Router was considered "_not ready for production,_" but with our experience and the Next.js team now blessing it as _stable_, we encourage you to try it out in your project! We do recommend taking the time to read through the [official docs](https://nextjs.org/docs) thoroughly. With a number of changes from the Pages Directory, you just need to dig into the docs and give it a try.

Overall, we’re happy with our decision to be one of the early production apps adopting the App Router. With a major re-write, it was our chance to use the cutting-edge to avoid another re-write or upgrade again in the near future.

In an upcoming post, we’ll discuss how debugging has changed with the introduction of React Server Components, how to handle cold boots, and more.

If you’re curious to see our new app in action, check out our new Inngest Dashboard by signing up [here](/sign-up?ref=blog-5-lessons-learned-from-taking-next-js-app-router-to-production).

![The Inngest Dashboard](/assets/blog/5-lessons-learned-from-taking-next-js-app-router-to-production/inngest-dashboard.png)
