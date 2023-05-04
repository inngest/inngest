---
focus: true
heading: "Is the Next.js 13 App Router Stable Enough for Production?"
subtitle: What have we learned from building our new production app with the still in beta Next.js 13 App Router?
image: "/assets/blog/is-the-next.js-13-app-router-stable-enough-for-production/featured-image.webp"
date: 2023-05-04
author: Igor Gassmann
---

Next.js 13 introduced the new [App Router](https://beta.nextjs.org/docs#introducing-the-app-router) that offers several new features, including [Nested Layouts](https://beta.nextjs.org/docs/routing/pages-and-layouts#nesting-layouts), [Server Components](https://beta.nextjs.org/docs/rendering/server-and-client-components#server-components), and [Streaming](https://beta.nextjs.org/docs/data-fetching/streaming-and-suspense). It's the first open-source implementation that allows developers to fully leverage the primitives brought by [React 18](https://react.dev/blog/2022/03/29/react-v18).

That being said, the App Router is currently still in beta, and the Next.js team does not recommend using it in production. So why did we use it to build our new production app, and should you do the same? Here are the lessons we’ve learned shipping this in production.

## Why Did We Decide to Use the App Router?

We needed to redesign and rewrite the Inngest Dashboard to account for new features (branch deploys, metrics, logs, etc.) and improve the developer experience from our legacy setup. This was a greenfield project—no code from the previous app was reused, except for our existing GraphQL API.

We chose Next.js as a framework, but we still had to decide whether to use the established router from the `pages/` directory or the newer, but still in beta, App Router. We were thrilled by the new possibilities unlocked by its features but simultaneously concerned about the risks and limitations we would face.

The App Router has several advantages:

- Thanks to Nested Layouts, we could more easily share UI between routes while preserving state and avoiding expensive re-renders.
- We could use [React Server Components](https://beta.nextjs.org/docs/rendering/server-and-client-components#server-components) to better leverage the server.
- We could display parts of a page sooner, without waiting for all the data to load, thanks to [Streaming](https://beta.nextjs.org/docs/data-fetching/streaming-and-suspense).
- We’d be ready for the future; no need to migrate when the `pages/` directory would eventually become obsolete.

However, there were drawbacks:

- We’d have to learn new concepts and a new mental model brought by the App Router and React Server Components.
- Third-party tools still had limited support for it. e.g., [Sentry](https://github.com/getsentry/sentry-javascript/issues/6726), [Storybook](https://github.com/storybookjs/storybook/blob/next/code/frameworks/nextjs/README.md#stories-for-pagescomponents-which-fetch-data)
- There were few learning resources or examples of implementations.
- We couldn’t anticipate any unknown issues or pitfalls, being one of the first to use it.

Ultimately, we chose the App Router. Since this was a new app, we estimated that it wouldn’t have much complexity in its first months. This would reduce the risk of encountering any major issues or limitations. Since we were investing in a major rewrite, we wanted to use the cutting edge so we wouldn’t have to do another one in the future.

## Lessons Learned From Putting App Router Into Production

After a few weeks of development, we shipped our new app using the App Router. Here are some of the things we learned:

### Caching Is Still a Work-in-Progress

The App Router introduces two new caches:

- A [**client-side cache**](https://beta.nextjs.org/docs/routing/linking-and-navigating#client-side-caching-of-rendered-server-components) that stores the payload of previously navigated or prefetched routes to help make navigation feel near-instant.
- A [**server-side cache**](https://beta.nextjs.org/docs/data-fetching/fundamentals#caching-data) that improves the performance of server-side data fetching

We often found it hard to make sense of those caches and which cache invalidation rules they follow. We also encountered an [issue](https://github.com/vercel/next.js/issues/42991) with the client-side cache, and it seems the caching logic is still being [tweaked](https://github.com/vercel/next.js/pull/48383).

You must also remember to call [`router.refresh()`](https://beta.nextjs.org/docs/data-fetching/mutating) after each mutation to invalidate the cache and refresh the server-provided data. If you forget to do it, your Server Components will then display stale data.

If you aren’t making `GET` requests with the `fetch()` function when fetching on the server, the 
returned data will be considered static by Next.js unless you remember to use the [`cache()` function](https://beta.nextjs.org/docs/data-fetching/caching#per-request-caching). This means the returned data will be fetched at build time, cached, and reused on each request. You would then end up with stale data when dealing with actual dynamic data.

If you’re building an app that deals largely with dynamic data, we would recommend disabling the server-side cache for your route segments by setting `export const dynamic = 'force-dynamic'` in your `layout.tsx` and `page.tsx` files until you feel comfortable with the Next.js cache. That would help you with avoiding unexpected caching issues. Notice, though, that it doesn’t affect the client-side cache.

### Streaming Improved Page Loading Experience

The Streaming feature of the App Router allowed us to display parts of a page sooner without waiting for all the data to load.

For example, we were able to stream the initial content of a page, such as the header and navigation menu, before all the data had finished loading. This allowed users to start interacting with the page sooner, even if some content was still loading, leading to an improved user experience.

<video controls src="/assets/blog/is-the-next.js-13-app-router-stable-enough-for-production/streaming-in-action.mp4" />

### URL Search Parameters Can’t Be Used in a Layout Server Component

[Unlike page components](https://beta.nextjs.org/docs/api-reference/file-conventions/page#searchparams-optional), the App Router doesn’t make URL search parameters (`?key1=value1&key2=value2`) available to layout server components. This is because a [layout component](https://beta.nextjs.org/docs/api-reference/file-conventions/layout) is not re-rendered when the user navigates to a different page within that same layout. The search parameters could change between navigations, leading to the layout component having outdated values for the search parameters. The router works that way to provide faster navigation.

However, this was an issue for us since we wanted to implement an optional global filter for our app that would persist in the URL. We allow users to select an environment that filters all displayed data (functions, events, and deploys) to that environment.

![The environment selector in the Inngest Dashboard](/assets/blog/is-the-next.js-13-app-router-stable-enough-for-production/environment-selector.png)

Our first solution was to add an `env` search parameter to the URL, like this: `https://app.inngest.com/functions?env=staging`. However, we soon discovered this didn’t work for our layout components that needed to fetch data based on the selected environment.

To solve this issue, we had to convert the search parameter into an URL parameter: `https://app.inngest.com/env/staging/functions`. Layout components can receive [dynamic route parameters](https://beta.nextjs.org/docs/api-reference/file-conventions/layout#params-optional), resolving our problem. Since the App Router primarily uses paths for routing, we found that it works best when putting parameters like this into the URL path.

### The Opinionated File Structure Brings Many Benefits

The App Router is configured by creating [special files](https://beta.nextjs.org/docs/routing/fundamentals#file-conventions) within a folder structure. You can use those special files to declare [Suspense](https://react.dev/reference/react/Suspense) and [Error](https://react.dev/reference/react/Component#catching-rendering-errors-with-an-error-boundary) boundaries at multiple nesting levels.

![A section of the Inngest Dashboard’s file structure](/assets/blog/is-the-next.js-13-app-router-stable-enough-for-production/file-structure.png)

We can understand our app's structure and see at which levels the suspense and error boundaries are declared just by looking at our file structure. Before the App Router, we would have to look inside components to find these boundaries. This also has the benefit of promoting the use of those React primitives, which are often neglected by React developers.

Additionally, we can now [colocate](https://beta.nextjs.org/docs/routing/fundamentals#colocation) our files with our routes, such as components, tests, and styles. This is especially useful for files that are only used by one route.

### **Learning Curve and Limited Learning Resources**

The steep learning curve was one of the biggest challenges we faced with the App Router. There's a lot to learn between the new routing, React Server Components, and caches. React Server Components also requires us to update our existing mental models for how to structure components which can be challenging when you’ve been building react apps for years only on the client. This learning curve undoubtedly slowed down our development process, and we’re still learning.

The official Next.js team did excellent work with their [docs](https://beta.nextjs.org/docs), and it is immensely helpful to learn the basics. However, as soon as you need to implement something beyond the typical path, you will likely struggle to find resources such as blog posts or implementation examples to help you. You’ll often be more successful in these moments by looking at [GitHub issues](https://github.com/vercel/next.js/issues) and Twitter conversations.

At this moment, we still have some open questions that we’re trying to figure out:

- How to properly dedupe fields across multiple GraphQL queries when using React Server Components?
- How do you add pagination to a list in a layout server component?

With time, we believe these challenges will be resolved, and we’ll see more learning resources and best practices emerging from the community, such as this blog post. But until then, it’s important to be patient and persistent in seeking solutions. It’s also helpful to share our experiences and solutions with others to help build a more substantial knowledge base.

Considering those challenges, we recommend falling back to a client component when you get stuck trying to implement something with a React Server Component. In Next.js, client components still benefit from being [pre-rendered](https://beta.nextjs.org/docs/rendering/server-and-client-components#client-components) on the server, like in the `pages/` directory.

## **Conclusion**

Overall, we found that the App Router is now stable enough if you’re building a new app from the ground up, but it requires significant investment in learning and experimentation. We wouldn’t recommend yet migrating an existing app since you’re more likely to encounter issues when trying to support all your specific use cases.

If you’re considering using the App Router for your project, we would recommend taking the time to read through the [official docs](https://beta.nextjs.org/docs) thoroughly. If you neglect this step, you may cause yourself problems. With the right approach and mindset, the App Router can be a powerful tool for building complex and flexible web applications.

In a future post, we’ll discuss how failing to declare Suspense boundaries properly can make your app feel slow, how debugging has changed with the introduction of React Server Components, and how to handle cold boots.

If you’re curious to see the App Router in action, check out our new Inngest Dashboard by signing up [here](/sign-up?ref=blog-is-the-next.js-13-app-router-stable-enough-for-production).

![The Inngest Dashboard](/assets/blog/is-the-next.js-13-app-router-stable-enough-for-production/inngest-dashboard.png)
