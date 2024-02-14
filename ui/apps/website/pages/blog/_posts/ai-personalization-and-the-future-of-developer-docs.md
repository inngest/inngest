---
heading: "AI Personalization and the Future of Developer Docs"
subtitle: "Providing developer-specific examples to help developers learn how to use the Inngest SDK. The beginning of AI-personalized learning flows for users."
image: /assets/blog/openai-durable-functions-with-inngest/featured-image.png
date: 2023-02-09
author: Jack Williams
disableCTA: true
---

OpenAI Codex is the model that powers GitHub Copilot, an “AI pair programmer” that suggests code in your IDE based on your codebase and billions of lines of open-source repositories.

Today, we’re leveraging that power to release an AI workflow builder, so you can instantly understand how your problem might be solved with Inngest by simply explaining what you’d like to achieve.

<aside>
  <Button href="/ai-personalized-documentation" arrow="right">
    <span className="text-white">Try it out now</span>
  </Button>
</aside>

## Imagination-driven documentation

Most platform documentation deals with directly or indirectly teaching the reader concepts embedded in that platform’s DNA. Often—and especially in the case of platforms with SDKs such as Inngest—this is achieved using code examples as the major talking point. These can, however, be difficult to parse depending on the reader.

These written examples, like most documentation, are static; one key example must serve many readers, and those readers must make an extra jump in their heads to attribute the code they’re reading and the concept they are learning to the final problem they’re trying to solve.

When communicating verbally, it’s much easier to give somebody an understanding of a concept by relating it to the one piece of the puzzle they already know. Similarly, by generating code upfront, the reader can apply their specific context to the platform straight away without that second hop.

![Examples of prompts given to the bot to generate code](/assets/blog/openai-durable-functions-with-inngest/prompt-examples.png)

## Challenges of building with AI

As AI's immediate practical applications grow and grow, developers are rushing to leverage the technology where they can. Utilising the pressured current services available, however, isn't an easy ride.

- Responses can be exceedingly slow (over a minute!)
- Timeouts or other errors can cause requests to fail altogether
- Limited input sizes require splitting work in to multiple requests

This is not dissimilar to the handling of any API, though most requirements for reliably working with services such as OpenAI use tooling outside that of common serverless platforms.

We've been watching and helping developers leverage Inngest to achieve reliable AI usage with OpenAI and other such services. Our strong ties with serverless platforms such as [Vercel](https://vercel.com/integrations/inngest) (check out our [Quick start tutorial](/docs) to get started) means that these complex, reliable flows can exist in your codebase with no infrastructure requirements whatsoever.

## How the bot is built

The OpenAI piece of the bot is wonderfully simple, hosted on [Deno Deploy](https://deno.com/deploy).

1. User submits a prompt
2. Merge the user’s prompt with a prompt engineered to teach Codex details of the Inngest SDK
3. Parse the result and pull out code, description, and references used

We’ve published the result over at [github.com/inngest/inngestabot](http://github.com/inngest/inngestabot) for you to go and have a look at the code if you fancy.

### Discord bot integration

In addition to being available on the website, the bot is available for work on our [community Discord server](https://www.inngest.com/discord), where it dutifully awaits the opportunity to create more functions.

You can ask it to create a function by simply tagging it in a message, like so:

> **_@inngestabot Send a welcome email to a new user._**

![Example prompt to Inngest discord bot to generate durable workflow that sends a welcome email to a new user](/assets/blog/openai-durable-functions-with-inngest/discord-message.gif)

Slash commands for Discord bots still rely on long-lived, serverful connections, for example to listen to incoming messages as required here. To handle this event in a serverless Inngest function required a workaround.

For this, we placed a tiny piece of Deno code on [Fly.io](http://Fly.io) using [Discordeno](https://deno.land/x/discordeno) that boots up a bot and emits an Inngest event whenever a request to create a function is received.

```ts
import { createBot, Inngest } from "./deps.ts";

// Create the Discord bot
const bot = createBot({
  token: Deno.env.get("DISCORD_TOKEN"),
});

// Create an Inngest instance
const inngest = new Inngest({ name: "Discord Bot" });

bot.events.messageCreate = async (_b, message) => {
  // Check if the message is a request, then...
  await inngest.send("inngestabot/message.received", {
    data: {
      message: {
        channelId: message.channelId.toString(),
        content: message.content,
        id: message.id.toString(),
      },
    },
    user: { authorId: message.authorId.toString() },
  });
};
```

We then have a tiny Inngest function, again hosted on [Deno Deploy](https://deno.com/deploy) using our [Deno integration](https://www.inngest.com/docs/sdk/serve#framework-fresh-deno), which handles these messages in the background, generating replies and sending them back to Discord. OpenAI responses can be slow, and the Discord bot can get busy, so it’s critical to pull this work to somewhere more scalable using Inngest.

```ts
inngest.createFunction(
  { name: "Handle Inngestabot message" },
  { event: "inngestabot/message.received" },
  async ({ event, step }) => {
    const { message } = event.data;

    // Generate a reply using our OpenAI Codex endpoint
    // OpenAI can sometimes error — `step.run` automatically retries on errors
    const reply = await step.run("Generate reply from OpenAI", async () => {
      const res = await fetch(OPENAI_ENDPOINT, {
        method: "POST",
        body: JSON.stringify({ message: message.content }),
      });

      return await res.json();
    });

    // Parse and send the reply to Discord
    await step.run("Send reply to Discord", async () => {
      return await bot.sendMessage(
        message.channelId,
        createDiscordMessageFromReply(reply)
      );
    });
  }
);
```

We make sure to use Inngest’s step tooling here to provide retries to the reply generation and sending the Discord message. We go a bit deeper in the final code, too, also ensuring the bot adds a thinking reaction (🤔) to show that it’s on the case - make sure to check out the [inngest/inngestabot](https://github.com/inngest/inngestabot) repo.

## Future

AI’s here to stay, and more obvious patterns are emerging for how it can help us. There are fantastic examples of AI helping us learn already being demonstrated, such as [Supabase's Clippy](https://supabase.com/blog/chatgpt-supabase-docs?ref=inngest), [Astro's Houston](https://houston.astro.build/) and [Dagster's support bot](https://dagster.io/blog/chatgpt-langchain).

Even specific to just software, personalizing learning experiences will accelerate the pace at which we can build and communicate concepts across the entire community; examples such as this are just the beginning.

Make sure to check out Inngest’s offering and see how we might be able to solve your problem today.

- [Try it out now on our site →](/ai-personalized-documentation) (no Discord needed)
- [Introduction to Inngest](/docs)
- Check out the [inngest/inngestabot](https://github.com/inngest/inngestabot) code
