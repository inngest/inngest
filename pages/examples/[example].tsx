import styled from "@emotion/styled";
import fetch from "cross-fetch";
import shuffle from "lodash.shuffle";
import { marked } from "marked";
import { GetStaticPaths, GetStaticProps } from "next";
import Button from "src/shared/Button";
import { CommandSnippet } from "src/shared/CommandSnippet";
import Footer from "src/shared/footer";
import Github from "src/shared/Icons/Github";
import Nav from "src/shared/nav";
import { z } from "zod";

const miniCache: Record<string, any> = {};

const treeSchema = z.object({
  path: z.string(),
  sha: z.string().or(z.null()),
  type: z.union([z.literal("blob"), z.literal("tree"), z.literal("commit")]),
  url: z.string(),
});

const blobSchema = z.object({
  content: z.string(),
  encoding: z.union([z.literal("utf-8"), z.literal("base64")]),
});

interface Props {
  id: string;
  name: string;
  description?: string;
  readme?: string;
  examples: Awaited<ReturnType<typeof getExamples>>;
}

export default function LibraryExamplePage(props: Props) {
  return (
    <div>
      <Nav sticky nodemo />
      <div className="container mx-auto pt-32 pb-24 flex flex-row">
        <div className="text-center px-6 max-w-4xl mx-auto flex flex-col space-y-6">
          <h1>{props.name}</h1>
          <p className="subheading">{props.description}</p>
          <CommandSnippet
            command={`npx inngest-cli init --template github.com/inngest/inngest#examples/${props.id}`}
            copy
          />
          <div className="flex flex-row justify-center">
            <Button kind="primary" href={`/examples?ref=examples/${props.id}`}>
              See more examples
            </Button>
            <Button
              kind="outline"
              href={`https://github.com/inngest/inngest/tree/main/examples/${props.id}`}
              target="_blank"
              className="flex flex-row items-center justify-center space-x-1"
            >
              <Github />
              <div>Explore the code</div>
            </Button>
          </div>
        </div>
      </div>

      <div className="pb-24 flex items-center justify-center">
        {props.readme ? (
          <div
            className="prose"
            dangerouslySetInnerHTML={{ __html: props.readme }}
          />
        ) : null}
      </div>
      <Highlights>
        <div className="container mx-auto p-12 grid grid-cols-3 gap-12">
          {shuffle(props.examples)
            .slice(0, 3)
            .map((example) => (
              <div>
                {example.name} - {example.description}
              </div>
            ))}
        </div>
      </Highlights>
      <Footer />
    </div>
  );
}

export const getStaticPaths: GetStaticPaths = async () => {
  const examples = await getExamples();

  const paths = examples.map((example) => ({
    params: { example: example.id },
  }));

  return { paths, fallback: false };
};

export const getStaticProps: GetStaticProps = async (ctx) => {
  const examples = await getExamples();
  const example = examples.find(({ id }) => id === ctx.params?.example);

  if (!example) {
    throw new Error("Could not find example");
  }

  let readme:
    | string
    | null = `Inngest is an open-source, event-driven platform which makes it easy for developers to build, test, and deploy serverless functions without worrying about infrastructure, queues, or stateful services.\n\nUsing Inngest, you can write and deploy serverless step functions which are triggered by events without writing any boilerplate code or infra. Learn more at https://www.inngest.com.\n\n- [Overview](#overview)\n- [Quick Start](#quick-start)\n- [Project Architecture](#project-architecture)\n- [Community](#community)\n\n<br />\n\n## Overview\n\nInngest makes it simple for you to write delayed or background jobs by triggering functions from events â\x80\x94 decoupling your code from your application.\n\n- You send events from your application via HTTP (or via third party webhooks, e.g. Stripe)\n- Inngest runs your serverless functions that are configured to be triggered by those events, either immediately, or delayed.\n\nInngest abstracts the complex parts of building a robust, reliable, and scalable architecture away from you so you can focus on writing amazing code and building applications for your users.\n\nWe created Inngest to bring the benefits of event-driven systems to all developers, without having to write any code themselves. We believe that:\n\n- Event-driven systems should be _easy_ to build and adopt\n- Event-driven systems are better than regular, procedural systems and queues\n- Developer experience matters\n- Serverless scheduling enables scalable, reliable systems that are both cheaper and better for compliance\n\n[Read more about our vision and why this project exists](https://www.inngest.com/blog/open-source-event-driven-queue)\n\n<br />\n\n## Quick Start\n\n1. Install the Inngest CLI to get started:\n\n\`\`\`bash\ncurl -sfL https://cli.inngest.com/install.sh | sh \\\n  && sudo mv ./inngest /usr/local/bin/inngest\n# or via npm\nnpm install -g inngest-cli\n\`\`\`\n\n2.  Create a new function. It will prompt you to select a programming language and what event will trigger your function. Optionally use the \`--trigger\` flag to specify the event name:\n\n\`\`\`shell\ninngest init --trigger demo/event.sent\n\`\`\`\n\n3. Run your new hello world function with dummy data:\n\n\`\`\`shell\ninngest run\n\`\`\`\n\n4. Run the Inngest DevServer. This starts a local "Event API" which can receive events. When events are received, functions with matching triggers will automatically be run. Optionally use the \`-p\` flag to specify the sport for the Event API.\n\n\`\`\`shell\ninngest dev -p 9999\n\`\`\`\n\n5. Send events to the DevServer. Send right from your application using HTTP + JSON or simply, as a curl with a dummy key of \`KEY\`.\n\n\`\`\`shell\ncurl -X POST --data \'{"name":"demo/event.sent","data":{"test":true}}\' http://127.0.0.1:9999/e/KEY\n\`\`\`\n\nThat\'s it - your hello world function should run automatically! When you \`inngest deploy\` your function to Inngest Cloud or your self-hosted Inngest. Here are some more resources to get you going:\n\n- [Full Quick Start Guide](https://www.inngest.com/docs/quick-start?ref=github)\n- [Function arguments & responses](https://www.inngest.com/docs/functions/function-input-and-output?ref=github)\n- [Sending Events to Inngest](https://www.inngest.com/docs/event-format-and-structure?ref=github)\n- [Inngest Cloud: Managing Secrets](https://www.inngest.com/docs/cloud/managing-secrets?ref=github)\n- [Self-hosting Inngest](https://www.inngest.com/docs/self-hosting?ref=github)\n\n<br />\n\n## Project Architecture\n\nFundamentally, there are two core pieces to Inngest: _events_ and _functions_. Functions have several sub-components for managing complex functionality (eg. steps, edges, triggers), but high level an event triggers a function, much like you schedule a job via an RPC call to a queue. Except, in Inngest, **functions are declarative**. They specify which events they react to, their schedules and delays, and the steps in their sequence.\n\n<br />\n\n<p align="center">\n  <img src=".github/assets/architecture-0.5.0.png" alt="Open Source Architecture" width="660" />\n</p>\n\nInngest\'s architecture is made up of 6 core components:\n\n- **Event API** receives events from clients through a simple POST request, pushing them to the **message queue**.\n- **Event Stream** acts as a buffer between the **API** and the **Runner**, buffering incoming messages to ensure QoS before passing messages to be executed.<br />\n- A **Runner** coordinates the execution of functions and a specific runâ\x80\x99s **State**. When a new function execution is required, this schedules running the functionâ\x80\x99s steps from the trigger via the **executor.** Upon each stepâ\x80\x99s completion, this schedules execution of subsequent steps via iterating through the functionâ\x80\x99s **Edges.**\n- **Executor** manages executing the individual steps of a function, via _drivers_ for each stepâ\x80\x99s runtime. It loads the specific code to execute via the **DataStore.** It also interfaces over the **State** store to save action data as each finishes or fails.\n  - **Drivers** run the specific action code for a step, eg. within Docker or WASM. This allows us to support a variety of runtimes.\n- **State** stores data about events and given function runs, including the outputs and errors of individual actions, and whatâ\x80\x99s enqueued for the future.\n- **DataStore** stores persisted system data including Functions and Actions version metadata.\n- **Core API** is the main interface for writing to the DataStore. The CLI uses this to deploy new funtions and manage other key resources.\n\nAnd, in this CLI:\n\n- The **DevServer** combines all of the components and basic drivers for each into a single system which loads all functions on disk, handles incoming events via the API and executes functions, all returning a readable output to the developer. (_Note - the DevServer does not run a Core API as functions are loaded directly from disk_)\n\nTo learn how these components all work together, [check out the in-depth architecture doc](To learn how these components all work together, [check out the in-depth architecture doc](/docs/ARCHITECTURE.md). For specific information on how the DevServer works and how it compares to production [read this doc](/docs/DEVSERVER_ARCHITECTURE.md).\n).\n\n<br />\n\n## Community\n\n- [**Join our online community for support, to give us feedback, or chat with us**](https://www.inngest.com/discord).\n- [Post a question or idea to our Github discussion board](https://github.com/orgs/inngest/discussions)\n- [Read the documentation](https://www.inngest.com/docs)\n- [Explore our public roadmap](https://github.com/orgs/inngest/projects/1/)\n- [Follow us on Twitter](https://twitter.com/inngest)\n- [Join our mailing list](https://www.inngest.com/mailing-list) for release notes and project updates\n\n## Contributing\n\nWeâ\x80\x99re excited to embrace the community! Weâ\x80\x99re happy for any and all contributions, whether theyâ\x80\x99re feature requests, ideas, bug reports, or PRs. While weâ\x80\x99re open source, we donâ\x80\x99t have expectations that people do our work for us â\x80\x94 so any contributions are indeed very much appreciated. Feel free to hack on anything and submit a PR.\n`;
  const readmeNode = example.tree.find(
    ({ path, type }) => path === "README.md" && type === "blob"
  );

  if (readmeNode?.url) {
    const readmeRaw = await reqWithSchema(readmeNode.url, blobSchema);

    readme =
      (readmeRaw.encoding === "base64"
        ? Buffer.from(readmeRaw.content, "base64").toString()
        : readmeRaw.content
      ).trim() || null;
  }

  // TODO Moev this to the if; just testing here
  readme = marked.parse(readme);

  return {
    props: {
      ...example,
      examples,
      readme,
      meta: {
        title: `Example: ${example.name}`,
        description: example.description || "Some default meta description",
      },
    },
  };
};

const reqWithSchema = async <T extends z.ZodTypeAny>(
  url: string,
  schema: T
): Promise<z.output<T>> => {
  const json =
    miniCache[url] ||
    (await fetch(url, {
      headers: {
        "User-Agent": "inngest",

        /**
         * If a `GITHUB_TOKEN` env var is available, use it here in order to
         * increase the rate limit allowance.
         *
         * GitHub Action runners get 1,000 requests per hour. A single deploy
         * here uses a maximum of (3 + 3n) requests, where n is the number of
         * examples.
         *
         * e.g. a single deploy of 3 examples can use up to a total of 9
         * requests.
         *
         * Using a set token here allows us up to 15,000 requests per hour. If
         * we hit that limit, we can add a pre-build step to clone the repo and
         * change how this data is fetched.
         */
        Authorization: process.env.GITHUB_TOKEN
          ? `token ${process.env.GITHUB_TOKEN}`
          : "",
      },
    })
      .then((res) => res.json())
      .then((data) => {
        miniCache[url] = data;
        return data;
      }));

  try {
    return schema.parse(json);
  } catch (err) {
    console.error("Error reading json:", json, err);
    throw err;
  }
};

export const getExamples = async () => {
  const latestCommit = await reqWithSchema(
    "https://api.github.com/repos/inngest/inngest/commits/main",
    z.object({
      commit: z.object({ tree: z.object({ url: z.string() }) }),
    })
  );

  const commitContents = await reqWithSchema(
    latestCommit.commit.tree.url,
    z.object({
      tree: z.array(treeSchema),
    })
  );

  const examplesPath = commitContents.tree.find(
    ({ path, type }) => path === "examples" && type === "tree"
  );

  if (!examplesPath?.url) {
    throw new Error("Could not find examples path");
  }

  const examplesTree = await reqWithSchema(
    examplesPath.url,
    z.object({
      tree: z.array(treeSchema),
    })
  );

  const exampleNodes = examplesTree.tree.filter(({ type }) => type === "tree");

  const examples = await Promise.all(
    exampleNodes.map(async (node) => {
      const example = await reqWithSchema(
        node.url,
        z.object({ tree: z.array(treeSchema) })
      );

      const inngestJsonNode = example.tree.find(
        ({ path, type }) => path === "inngest.json" && type === "blob"
      );

      if (!inngestJsonNode?.url) {
        throw new Error("Could not find inngest.json in example");
      }

      const inngestJsonRaw = await reqWithSchema(
        inngestJsonNode.url,
        blobSchema
      );

      return z
        .object({
          id: z.string(),
          tree: z.array(treeSchema),
          name: z.string(),
          description: z.string().optional(),
        })
        .parse({
          ...JSON.parse(
            inngestJsonRaw.encoding === "base64"
              ? Buffer.from(inngestJsonRaw.content, "base64").toString()
              : inngestJsonRaw.content
          ),
          id: node.path,
          tree: example.tree,
        });
    })
  );

  return examples;
};

const Highlights = styled.div`
  background: var(--bg-color-d);
  color: #fff;
`;
