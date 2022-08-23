import styled from "@emotion/styled";
import shuffle from "lodash.shuffle";
import { marked } from "marked";
import { GetStaticPaths, GetStaticProps } from "next";
import Link from "next/link";
import Script from "next/script";
import { useEffect, useMemo, useState } from "react";
import Button from "src/shared/Button";
import { CommandSnippet } from "src/shared/CommandSnippet";
import Footer from "src/shared/footer";
import Github from "src/shared/Icons/Github";
import Nav from "src/shared/nav";
import { reqWithSchema } from "src/utils/fetch";
import { blobSchema, getExamples } from ".";
import hljs from "highlight.js";

interface Props {
  id: string;
  name: string;
  description?: string;
  readme?: string;
  examples: Awaited<ReturnType<typeof getExamples>>;
}

export default function LibraryExamplePage(props: Props) {
  const [loadingMermaid, setLoadingMermaid] = useState(true);

  useEffect(() => {
    // @ts-ignore
    if (typeof mermaid !== "undefined") {
      // @ts-ignore
      mermaid?.contentLoaded();
    }
  }, [loadingMermaid, props.readme]);

  const nextExamples = useMemo(() => {
    return shuffle(props.examples)
      .filter(({ id }) => id !== props.id)
      .slice(0, 3);
  }, [props.examples]);

  return (
    <div>
      <Script
        src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"
        onLoad={() => {
          // Ignoring as we don't have (or really need) global typing for
          // mermaid.
          // @ts-ignore
          mermaid.initialize({ startOnLoad: true });

          // Set this to force a re-render
          setLoadingMermaid(false);
        }}
      />
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
            <Button
              kind="primary"
              href={`/quick-starts?ref=quick-starts/${props.id}`}
            >
              See more quickstarts
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
      {props.readme ? (
        <div className="mx-auto max-w-2xl pb-24 flex items-center justify-center">
          <div
            className="prose px-6 prose-a:text-[#5d5fef] max-w-none w-full"
            dangerouslySetInnerHTML={{ __html: props.readme }}
          />
        </div>
      ) : null}
      {nextExamples.length ? (
        <Highlights>
          <div className="container mx-auto p-12">
            <h2>More quickstarts</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-12 mt-4">
              {nextExamples.map((example) => (
                <div>
                  <Link
                    key={example.id}
                    href={`/quick-starts/${example.id}?ref=quick-starts/${props.id}`}
                    passHref
                  >
                    <a className="rounded-lg border border-gray-200 p-6 flex flex-col space-y-2 bg-black transition-all transform hover:scale-105 hover:shadow-lg">
                      <div className="font semi-bold text-white">
                        {example.name}
                      </div>
                      {example.description ? (
                        <div className="font-xs text-gray-200">
                          {example.description}
                        </div>
                      ) : null}
                      <a className="text-blue-500 font-semibold text-right">
                        Explore →
                      </a>
                    </a>
                  </Link>
                </div>
              ))}
            </div>
            <div className="w-full flex items-center justify-center mt-10">
              <Button
                kind="primary"
                href={`/quick-starts?ref=quick-starts/${props.id}`}
              >
                See all quickstarts
              </Button>
            </div>
          </div>
        </Highlights>
      ) : props.readme ? (
        <div className="w-full flex items-center justify-center">
          <Button
            kind="primary"
            href={`/quick-starts?ref=quick-starts/${props.id}`}
          >
            See all quickstarts
          </Button>
        </div>
      ) : null}
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
    | null = `# Example Smart Drip Activation Email Campaign\n\nThis example Inngest function defines a drip campaign that sends a new user a **targeted**\nemail _if_ they have _not_ already performed the action that the email is suggesting they do.\nImagine this flow for an application:\n\n\`\`\`mermaid\ngraph LR\nSource[Your app] -->|app/user.signup event| Inngest(Inngest)\nInngest --> Day(app/reservation.booked<br>received or 1 day elapsed)\nDay -->|app/reservation.booked| Nothing(Do nothing)\nDay -->|1 day elapsed| Email[Send email]\n\nclassDef inngest fill:#fff,stroke:#4636f5\nclassDef user fill:#4636f5,color:white,stroke:#4636f5\n\nclass Source,Email user\nclass Inngest,Day,Nothing inngest\n\`\`\`\n\n1. User signed up\n2. If the user _**has not**_ activated within 1 day:\n   - ➡️ Send them an email to guide them to take that action\n3. If the user _**has**_ activated:\n   - 👍 Don't do anything - They already figured it out!\n\n## Guide\n\n- [Function configuration](#function-configuration)\n- [Function code](#function-code)\n- [Sending events from your app](#sending-events-from-your-app)\n- [Deploying to Inngest Cloud](#deploying-to-inngest-cloud)\n\n### Function configuration\n\nThe function definition is annotated to show how the above is defined in config:\n\n1. After the \`app/user.signup\` event is received...\n2. Wait up to 1 day (\`1d\`) for the user to activate (\`app/reservation.booked\`)...\n3. If they have not triggered the activation event (\`app/reservation.booked\`)...\n4. Send an email via Sendgrid\n\n\`\`\`json\n{\n  "name": "activation-drip-email",\n  "id": "exciting-dogfish-63761d",\n  "triggers": [\n    {\n      // When this event is received by Inngest, it will start the function\n      "event": "app/user.signup",\n      "definition": {\n        "format": "cue",\n        // The file that declares the event schema that your app will send to Inngest\n        "def": "file://./events/app-user.signup.cue"\n      }\n    }\n  ],\n  "steps": {\n    "step-1": {\n      // This step will only be run "after" the below condition is true\n      "id": "step-1",\n      // This is the directory where your code will be including it's Dockerfile\n      "path": "file://./steps/1d-send-email",\n      "name": "activation-drip-email",\n      "runtime": {\n        "type": "docker"\n      },\n      // The "after" block lists conditions that will trigger the step to be run\n      "after": [\n        {\n          // "$trigger" means this will happen directly after the above event\n          // trigger: "app/user.signup"\n          "step": "$trigger",\n          // This is an asynchronous condition that will wait up to 1 day (1d)\n          // for the "app/reservation.booked" asynchronous event to be received\n          // The "match" checks that both events (the initial "event" and the\n          // "async" event) contain the same user id ("external_id").\n          "async": {\n            "event": "app/reservation.booked",\n            "match": "async.user.external_id == event.user.external_id",\n            "ttl": "1d",\n            "onTimeout": true\n          }\n        }\n      ]\n    }\n  }\n}\n\`\`\`\n\n### Function code\n\nAll of the code for the function that sends the email to SendGrid, is\nwithin the \`steps/1d-send-email/src/index.ts\` file. This code will be passed\nthe \`app/user.signup\` event if the 1 day timeout has been reached before\nany \`app/reservation.booked\` email is received.\n\n➡️ [Check out \`index.ts\`](/steps/1d-send-email/src/index.ts)\n\n### Sending events from your app\n\nImagine a JavaScript application, using the [Inngest library](https://github.com/inngest/inngest-js#readme) in your \`/signup\` endpoint you can add the following code:\n\n\`\`\`js\nimport { Inngest } from "inngest";\n\n// POST myapp.com/signup\nexport default function signup(req, res) {\n  const user = await createUser(req.body.email, req.body.password);\n\n  // Send an event to Inngest\n  // You can get a Source Key from the sources section of the Inngest app\n  const inngest = new Inngest(process.env.INNGEST_SOURCE_API_KEY);\n  await inngest.send({\n    name: "app/user.signup",\n    data: { city: req.body.city /* e.g. "Detroit" */ },\n    user: {\n      external_id: user.id,\n      email: user.email,\n    },\n  });\n\n  res.redirect("/app")\n}\n\`\`\`\n\nAnd in your code that hands where the user is considered "activated", add the other event:\n\n\`\`\`js\nimport { Inngest } from "inngest";\n\n// POST myapp.com/bookReservation\nexport default function bookReservation(req, res) {\n  const user = await getUserFromSession(req)\n  const reservation = await createReservation(user, req.body.restaurantId, req.body.timestamp);\n\n  // Send an event to Inngest\n  const inngest = new Inngest(process.env.INNGEST_SOURCE_API_KEY);\n  await inngest.send({\n    name: "app/reservation.booked",\n    data: { restaurant: req.body.restaurantId },\n    user: {\n      external_id: user.id,\n    },\n  });\n\n  res.redirect("/app")\n}\n\`\`\`\n\n### Deploying to Inngest Cloud\n\nWith an [Inngest Cloud account created](https://inngest.com/sign-up?ref=github-example-drip), use the Inngest CLI to deploy your function:\n\n\`\`\`\nnpm install -g inngest-cli\ninngest login\ninngest deploy\n\`\`\`\n\nDone! Now send the events from your application and you'll see the events and function output in the Inngest web app.\n`;
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

  // Set a base URL for if using relative links in the README.md file
  const baseUrl = "https://raw.githubusercontent.com/inngest/inngest/main/";

  // TODO Moev this to the if; just testing here
  readme = marked.parse(
    readme
      .replaceAll(".github/assets/", `${baseUrl}/.github/assets/`)
      .replace(/(^#.*\n+)/, ""),
    {
      baseUrl: "https://raw.githubusercontent.com/inngest/inngest/main/",
      gfm: true,
      breaks: true,
      headerIds: true,
      renderer,
      highlight(code, lang) {
        const language = hljs.getLanguage(lang) ? lang : "plaintext";
        return hljs.highlight(code, { language }).value;
      },
    }
  );

  return {
    props: {
      ...example,
      examples,
      readme,
      meta: {
        title: `Quickstart: ${example.name}`,
        description:
          example.description ||
          `Get started using Inngest immediately with the ${example.name} quickstart.`,
      },
    },
  };
};

const Highlights = styled.div`
  background: var(--bg-color-d);
  color: #fff;
`;

const renderer = new marked.Renderer();
const defaultCodeRenderer = renderer.code.bind(renderer);
renderer.code = function (code, language) {
  if (language !== "mermaid") {
    return defaultCodeRenderer(code, language);
  }

  return `<div class="mermaid">${code}</div>`;
};
