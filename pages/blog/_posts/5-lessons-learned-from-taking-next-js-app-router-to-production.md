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

We recently redesigned the Inngest Dashboard to account for new features (branch deploys, metrics, logs, etc.). We rewrote it using Next.js 13 and the App Router. Here are the lessons we learned from building and shipping our new app with the Next.js App Router.

## 1. Caching Can Be Hard to Grasp

The App Router introduces two new caches:

- A [**client-side cache**](https://nextjs.org/docs/app/building-your-application/routing/linking-and-navigating#client-side-caching-of-rendered-server-components) that stores the payload of previously navigated or prefetched routes to help make navigation feel near-instant.
- A [**server-side cache**](https://nextjs.org/docs/app/building-your-application/data-fetching#caching-data) that improves the performance of server-side data fetching

We often found it hard to make sense of those caches, which cache invalidation rules they follow, and how they interact with each other. For example, we weren’t able to find information on the docs about how long the client-side cache is preserved. We had to dig into GitHub to find out that it’s preserved for [30 seconds](https://github.com/vercel/next.js/pull/48383).

If you aren’t making `GET` requests with the `fetch()` function when fetching on the server, the returned data will be considered static by Next.js unless you remember to use the [`cache()` function](https://nextjs.org/docs/app/building-your-application/data-fetching/caching#per-request-caching). This means the returned data will be fetched at build time, cached, and reused on each request. You would then end up with stale data when dealing with actual dynamic data.

If you’re building an app that deals largely with dynamic data, we would recommend disabling the server-side cache for your route segments by setting `export const dynamic = 'force-dynamic'` in your `layout.tsx` and `page.tsx` files until you feel comfortable with the Next.js cache. That would help you with avoiding unexpected caching issues. Notice, though, that it doesn’t affect the client-side cache.

## 2. Streaming Improved Page Loading Experience

The Streaming feature of the App Router allowed us to display parts of a page sooner without waiting for all the data to load.

For example, we were able to stream the initial content of a page, such as the header and navigation menu, before all the data had finished loading. This allowed users to start interacting with the page sooner, even if some content was still loading, leading to an improved user experience.

<video controls src="/assets/blog/5-lessons-learned-from-taking-next-js-app-router-to-production/streaming-in-action.mp4" />

## 3. URL Search Parameters Can’t Be Used in a Layout Server Component

[Unlike page components](https://nextjs.org/docs/app/api-reference/file-conventions/page#searchparams-optional), the App Router doesn’t make URL search parameters (`?key1=value1&key2=value2`) available to layout server components. This is because a [layout component](https://nextjs.org/docs/app/api-reference/file-conventions/layout) is not re-rendered when the user navigates to a different page within that same layout. The search parameters could change between navigations, leading to the layout component having outdated values for the search parameters. The router works that way to provide faster navigation.

However, this was an issue for us since we wanted to implement an optional global filter for our app that would persist in the URL. We allow users to select an environment that filters all displayed data (functions, events, and deploys) to that environment.

![The environment selector in the Inngest Dashboard](/assets/blog/5-lessons-learned-from-taking-next-js-app-router-to-production/environment-selector.png)

Our first solution was to add an `env` search parameter to the URL, like this: `https://app.inngest.com/functions?env=staging`. However, we soon discovered this didn’t work for our layout components that needed to fetch data based on the selected environment.

To solve this issue, we had to convert the search parameter into an URL parameter: `https://app.inngest.com/env/staging/functions`. Layout components can receive [dynamic route parameters](https://nextjs.org/docs/app/api-reference/file-conventions/layout#params-optional), resolving our problem. Since the App Router primarily uses paths for routing, we found that it works best when putting parameters like this into the URL path.

## 4. The Opinionated File Structure Brings Many Benefits

The App Router is configured by creating [special files](https://nextjs.org/docs/app/building-your-application/routing#file-conventions) within a folder structure. You can use those special files to declare [Suspense](https://react.dev/reference/react/Suspense) and [Error](https://react.dev/reference/react/Component#catching-rendering-errors-with-an-error-boundary) boundaries at multiple nesting levels.

![A section of the Inngest Dashboard’s file structure](/assets/blog/5-lessons-learned-from-taking-next-js-app-router-to-production/file-structure.png)

We can understand our app’s structure and see at which levels the suspense and error boundaries are declared just by looking at our file structure. Before the App Router, we would have to look inside components to find these boundaries. This also has the benefit of promoting the use of those React primitives, which are often neglected by React developers.

Additionally, we can now [colocate](https://nextjs.org/docs/app/building-your-application/routing#colocation) our files with our routes, such as components, tests, and styles. This is especially useful for files that are only used by one route.

## 5. Learning Curve and Limited Learning Resources

The steep learning curve was one of the biggest challenges we faced with the App Router. There’s a lot to learn between the new routing, React Server Components, and caches. React Server Components also requires us to update our existing mental models for how to structure components which can be challenging when you’ve been building react apps for years only on the client. This learning curve undoubtedly slowed down our development process, and we’re still learning.

The official Next.js team did excellent work with their [docs](https://nextjs.org/docs), and it is immensely helpful to learn the basics. However, as soon as you need to implement something beyond the typical path, you will likely struggle to find resources such as blog posts or implementation examples to help you. You’ll often be more successful in these moments by looking at [GitHub issues](https://github.com/vercel/next.js/issues) and Twitter conversations.

At this moment, we still have some open questions that we’re trying to figure out:

- How to properly dedupe fields across multiple GraphQL queries when using React Server Components?
- How do you add pagination to a list in a layout server component?

With time, we believe these challenges will be resolved, and we’ll see more learning resources and best practices emerging from the community, such as this blog post. But until then, it’s important to be patient and persistent in seeking solutions. It’s also helpful to share our experiences and solutions with others to help build a more substantial knowledge base.

Considering those challenges, we recommend falling back to a client component when you get stuck trying to implement something with a React Server Component. In Next.js, client components still benefit from being [pre-rendered](https://nextjs.org/docs/app/building-your-application/rendering#static-and-dynamic-rendering-on-the-server) on the server, like in the [Pages Directory](https://nextjs.org/docs/pages/building-your-application/rendering#pre-rendering).

## Conclusion

The Next.js App Router can provide a lot of benefits that enhance both end user and developer experience. If you choose to adopt it, you should consider that it’s still a relatively new technology that has a learning curve and aspects that are still in development.

We started building our app when the App Router was considered "_not ready for production,_" but with our experience and the Next.js team now blessing it as _stable_, we encourage you to try it out in your project! We do recommend taking the time to read through the [official docs](https://nextjs.org/docs) thoroughly. There are enough changes from the Pages Directory that can cause some considerable headaches if you don’t spend the time to understand it.

Overall, we’re happy with our decision to be one of the early production apps adopting the App Router. With a major re-write, it was our chance to use the cutting-edge to avoid another re-write or upgrade again in the near future.

In an upcoming post, we’ll discuss how debugging has changed with the introduction of React Server Components, how to handle cold boots, and more.

If you’re curious to see our new app in action, check out our new Inngest Dashboard by signing up [here](/sign-up?ref=blog-5-lessons-learned-from-taking-next-js-app-router-to-production).

![The Inngest Dashboard](/assets/blog/5-lessons-learned-from-taking-next-js-app-router-to-production/inngest-dashboard.png)
