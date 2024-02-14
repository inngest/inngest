import { GetStaticProps } from "next";
import Link from "next/link";
import Marquee from "react-fast-marquee";
import Check from "src/shared/Icons/Check";
import Header from "src/shared/Header";
import Container from "src/shared/layout/Container";
import PageContainer from "src/shared/layout/PageContainer";
import Logos from "src/shared/Home/Logos";
// Icons
import CopyBtn from "src/shared/Home/CopyBtn";
import { ChevronRightIcon } from "@heroicons/react/20/solid";
import CodeWindow from "src/shared/CodeWindow";
import Footer from "src/shared/Footer";
import CaseStudyCard from "src/shared/CaseStudy/CaseStudyCard";

export const getStaticProps: GetStaticProps = async () => {
  return {
    props: {
      designVersion: "2",
    },
  };
};

export default function AI() {
  return (
    <PageContainer>
      <Header />

      <AIHero />

      {/*
      <Container className="pt-4 pb-36">
        <p className="text-zinc-400 text-center">
          Companies of all sizes trust Inngest to power their AI functionality.
        </p>
      </Container>
      */}

      {/* Code example */}
      <Container>
        <h2
          className="
          text-4xl text-center font-bold mt-16 mb-2
        "
        >
          Focus on what matters: <span className="font-extrabold">AI</span>.
        </h2>
        <p className="text-center mb-20 opacity-60">
          Spend time developing what's important. Scale from day 0 by leaving
          the complex orchestration to us.
        </p>

        <div
          className="
          grid lg:grid-cols-3
          bg-slate-800/50 border-slate-700/30 rounded-lg border
          m-auto
          mt-8 mb-24
          hidden
          lg:grid
          lg:w-2/3
        "
        >
          <CodeWindow
            snippet={aiFlow}
            showLineNumbers={true}
            className="col-span-2 bg-transparent border-none"
          />
          {/*
          <div className="border-l border-slate-700/30 p-2">
            TODO: Flow diagram
          </div>
          */}
        </div>
      </Container>

      <AIScroll />

      {/* Call out box:  rapid development */}

      <Container className="pt-6">
        <GradientBox
          className="my-24 shadow-[0_10px_100px_0_rgba(52,211,153,0.2)]"
          border="2px"
        >
          <div
            className={`flex items-center justify-center bg-[#0a0a12] back rounded-t-md flex-col`}
          >
            <DevIcon className="mt-[-47px] mb-24" />

            <div className="flex flex-col items-center justify-center pb-24">
              <h2
                className="
                text-4xl text-center font-bold mb-8
              "
              >
                Rapidly iterate on complex AI chains,
                <br />
                directly&nbsp;in&nbsp;code.
              </h2>

              <p
                className="
                px-6
                lg:w-1/2
                text-center text-lg
              "
              >
                A simple, powerful interface that lets you define reliable flow
                control in your own code. Write AI workflows directly in your
                API using our SDK, with local testing out of the box.
              </p>

              <div
                className="
                flex
                flex-col lg:flex-row
                gap-8 pt-16 items-center justify-center
              "
              >
                <Link
                  href={`${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=ai-hero`}
                  className="rounded-md font-medium px-11 pr-9 py-3.5 bg-indigo-500 hover:bg-indigo-400 transition-all text-white whitespace-nowrap flex flex-row items-center
                  bg-emerald-400 text-[#050911]
                  "
                >
                  Get started{" "}
                  <ChevronRightIcon className="h-5 group-hover:translate-x-1 relative top-px transition-transform duration-150" />
                </Link>
              </div>
            </div>
          </div>

          <div className="grid lg:grid-cols-2 mt-[1px]">
            <div
              className="
                pt-14 pb-10 px-10 bg-[#0a0a12] flex flex-col items-center
                lg:rounded-bl-md lg:mr-[1px]
                sm:m-0
              "
            >
              <h3
                className="
                text-xl font-bold
                mb-4
                w-full
              "
              >
                Your data, your environment
              </h3>
              <p className="text-lg">
                Leverage data from your own database, vector store, or APIs
                directly in code — without complex interfaces or adapters.
              </p>

              <StoreLogos className="mt-12 opacity-90" />
            </div>
            <div
              className="
              py-14 px-10 bg-[#0a0a12] flex flex-col items-center
              mt-[1px]
              lg:mt-0
              lg:rounded-br-md
            "
            >
              <h3
                className="
                text-xl font-bold
                mb-4
                w-full
              "
              >
                Any AI model, any AI pattern
              </h3>
              <p className="text-lg">
                Easily implement any AI model as either a single call or using
                patterns like RAG, tree of thoughts, chain of thoughts, or
                safety rails, directly in code.
              </p>

              <ProviderLogos className="mt-14 opacity-90" />
            </div>
          </div>
        </GradientBox>
      </Container>

      <DevelopmentCopy />

      <Container>
        <GradientBox
          className="my-24 shadow-[0_10px_100px_0_rgba(52,211,153,0.2)] w-1/2 m-auto"
          border="0"
        >
          <div className="grid lg:grid-cols-2">
            <div
              className={`flex items-center justify-center bg-[#0a0a12] back flex-col p-8 text-center
              rounded
              m-[1px]
              lg:rounded-none lg:rounded-l-md`}
            >
              <span className="text-4xl">12x</span>
              <span>development speedup</span>
              <span className="text-xs opacity-40 mt-1">
                compared to traditional infrastructure
              </span>
            </div>
            <div
              className={`flex items-center justify-center bg-[#0a0a12] back flex-col m-[1px] p-8
              rounded
              lg:rounded-none lg:rounded-r-md lg:ml-0`}
            >
              <span className="text-4xl">75%</span>
              <span>total cost reduction</span>
              <span className="text-xs opacity-40 mt-1">
                on infrastructure and time spent
              </span>
            </div>
          </div>
        </GradientBox>
      </Container>

      <Container className="pt-28 pb-6">
        <ProdIcon className="m-auto" />

        <h2
          className="
          text-4xl text-center font-bold pt-20 pb-8
          w-2/3 m-auto
        "
        >
          Scale-ready productionAI in hours.
          Zero&nbsp;infrastructure&nbsp;required.
        </h2>

        <p className="text-lg text-slate-100 leading-8 w-1/2 text-center m-auto">
          Move to production by deploying Inngest functions inside your existing
          API, wherever it is — serverless, servers, or edge. Backed by rock
          solid external orchestration, your workflows are ready to scale in
          milliseconds.
        </p>
      </Container>

      <ProductionCopy />

      <div className="max-w-xl mx-auto">
        <CaseStudyCard
          href="/customers/aomni"
          logo="/assets/customers/aomni-logo.svg"
          name="Aomni"
          title="Productionizing AI-driven sales flows using serverless LLMs"
          snippet="Leveraging Inngest for production-grade complex state management and LLM chaining."
          tags={["AI"]}
        />
      </div>

      <p className="mt-8 text-zinc-400 text-center opacity-70 pt-16 mb-[-30px]">
        Use with any framework, on any cloud:
      </p>
      <Logos
        className="opacity-60 my-1 lg:my-1 pb-20"
        logos={[
          {
            src: "/assets/brand-logos/next-js-white.svg",
            name: "Next.js",
            href: "/docs/sdk/serve?ref=homepage-frameworks#framework-next-js",
          },
          {
            src: "/assets/brand-logos/express-js-white.svg",
            name: "Express.js",
            href: "/docs/sdk/serve?ref=homepage-frameworks#framework-express",
          },
          {
            src: "/assets/brand-logos/redwoodjs-white.svg",
            name: "RedwoodJS",
            href: "/docs/sdk/serve?ref=homepage-frameworks#framework-redwood",
          },
          {
            src: "/assets/brand-logos/remix-white.svg",
            name: "Remix",
            href: "/docs/sdk/serve?ref=homepage-frameworks#framework-remix",
          },
          {
            src: "/assets/brand-logos/deno-white.svg",
            name: "Deno",
            href: "/docs/sdk/serve?ref=homepage-frameworks#framework-fresh-deno",
          },
        ]}
      />

      <Footer />
    </PageContainer>
  );
}

const AIHero = () => (
  <Container>
    <div
      className="py-24 lg:py-48 gap-2 justify-between lg:items-center
      flex flex-col align-center
    "
    >
      <h1
        className="
        text-4xl font-bold leading-[48px] text-center
        sm:text-5xl sm:leading-[58px]
        lg:text-6xl lg:leading-[68px]
        tracking-[-2px]
        mb-8
        bg-gradient-to-r from-[#FFEAEA] to-[#D2CACF] drop-shadow bg-clip-text
        text-transparent
      "
      >
        Build powerful{" "}
        <span className="font-extrabold hero-text-shadow">AI workflows</span> in
        code.
      </h1>

      <p className="text-lg text-slate-100 leading-8 lg:w-1/2 text-center">
        Develop, test, and deploy reliable AI workflows to production with zero
        new infrastructure, in less than a day. Inngest’s event-driven workflows
        handle queueing, state, scale, and observability, letting you focus on
        what matters.
      </p>

      <div
        className="
        flex
        flex-col lg:flex-row
        gap-8 pt-16 items-center justify-center
      "
      >
        <Link
          href={`${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=ai-hero`}
          className="rounded-md font-medium px-11 py-3.5 bg-indigo-500 hover:bg-indigo-400 transition-all text-white whitespace-nowrap flex flex-row items-center
          bg-emerald-400 text-[#050911]
          "
        >
          Get started{" "}
          <ChevronRightIcon className="h-5 group-hover:translate-x-1 relative top-px transition-transform duration-150" />
        </Link>
        <Link
          href="/docs?ref=homepage-hero"
          className="group flex items-center gap-1 rounded-md px-11 py-3.5 bg-transparent transition-all text-indigo-100 border border-slate-800 hover:border-slate-600 whitespace-nowrap"
        >
          Read the docs
        </Link>
      </div>
    </div>
  </Container>
);

const AIScroll = () => {
  const list = [
    "RAG (Retrieval-Augmented Generation)",
    "Tree of Thoughts",
    "Embeddings",
    "Multi-model chains",
    "Guardrails",
    "Hallucination checking",
    "Observability",
    "Scoring",
    "Cost monitoring",
  ];
  return (
    <Marquee>
      <div className="font-mono text-sm text-zinc-500 py-8">
        {list.map((item, n) => (
          <span className="mx-14" key={n}>
            {item}
          </span>
        ))}
      </div>
    </Marquee>
  );
};

const GradientBox = ({ children, className = "", border = "2px" }) => (
  <div className={`mx-auto flex items-center justify-center ${className}`}>
    <div
      className={`w-full rounded-md bg-gradient-to-tl from-[#596555] via-[#D4FF8D] to-[#814828]`}
      style={{ padding: border }}
    >
      {children}
    </div>
  </div>
);

const DevelopmentCopy = () => {
  const handleCopyClick = (copy) => {
    navigator.clipboard?.writeText(copy);
  };
  return (
    <Container className="pb-8">
      <div className="grid lg:grid-cols-3 gap-20 pt-10 pb-20 px-10 opacity-80">
        <div>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width={36}
            height={36}
            viewBox="0 0 24 24"
            fill="none"
            className="my-6"
          >
            <path
              stroke="#fff"
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1}
              d="M14 15h7M3 15h2m0 0a2.5 2.5 0 1 0 5 0 2.5 2.5 0 0 0-5 0Zm15-6h1M3 9h7m6.5 2.5a2.5 2.5 0 1 1 0-5 2.5 2.5 0 0 1 0 5Z"
            />
          </svg>

          <h3
            className="
              text-xl font-semibold
              mb-4
              w-full
            "
          >
            Full flow control
          </h3>
          <p>
            Concurrency, rate limiting, debounce, automatic cancellation —
            everything you need to scale, while respecting rate limits, built in
            from the beginning.
          </p>
          <ul className="my-6 leading-8 opacity-70 leading-snug">
            <li className="flex items-center mb-4">
              <Check size={14} className="mr-2" /> Global and per-user
              concurrency limits
            </li>
            <li className="flex items-center mb-4">
              <Check size={14} className="mr-2" /> Per-user priorities with
              fairness guarantees
            </li>
            <li className="flex items-center mb-4">
              <Check size={14} className="mr-2" /> Auto-cancellation via events
              to save costs
            </li>
          </ul>
        </div>
        <div>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width={36}
            height={36}
            viewBox="0 0 24 24"
            fill="none"
            className="my-6 ml-[-2px]"
          >
            <path
              stroke="#fff"
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1}
              d="M17 15h-5m-5-5 3 2.5L7 15m-4 .8V8.2c0-1.12 0-1.68.218-2.108.192-.377.497-.682.874-.874C4.52 5 5.08 5 6.2 5h11.6c1.12 0 1.68 0 2.107.218.377.192.683.497.875.874.218.427.218.987.218 2.105v7.606c0 1.118 0 1.677-.218 2.104a2.003 2.003 0 0 1-.875.875c-.427.218-.986.218-2.104.218H6.197c-1.118 0-1.678 0-2.105-.218a2.001 2.001 0 0 1-.874-.875C3 17.48 3 16.92 3 15.8Z"
            />
          </svg>
          <h3
            className="
              text-xl font-semibold
              mb-4
              w-full
            "
          >
            Local development
          </h3>
          <p>
            Iterate on AI flows in your existing code base and test things
            locally using our dev server, with full production parity.
          </p>
          <ul className="my-6 leading-8 opacity-70 leading-snug">
            <li className="flex items-center mb-4">
              <Check size={14} className="mr-2" /> One-command setup for local
              dev
            </li>
            <li className="flex items-center mb-4">
              <Check size={14} className="mr-2" /> Visual workflow debugging and
              logs
            </li>
            <li className="flex items-center mb-4">
              <Check size={14} className="mr-2" /> Production parity for
              risk-free deploys
            </li>
          </ul>
        </div>
        <div>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            width={36}
            height={36}
            viewBox="0 0 24 24"
            className="my-6"
          >
            <path
              stroke="#fff"
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1}
              d="m15 7 5 5-5 5m-6 0-5-5 5-5"
            />
          </svg>
          <h3
            className="
              text-xl font-semibold
              mb-4
              w-full
            "
          >
            No magic: write regular code
          </h3>
          <p>
            Easily create AI workflows with regular code, using any library or
            integrations you need without learning anything new.
          </p>
          <ul className="my-6 leading-8 opacity-70 leading-snug">
            <li className="flex items-center mb-4">
              <Check size={14} className="mr-2" /> No constraints on what you
              can use
            </li>
            <li className="flex items-center mb-4">
              <Check size={14} className="mr-2" /> Simple, retryable steps using
              `step.run`
            </li>
            <li className="flex items-center mb-4">
              <Check size={14} className="mr-2" /> Use any library, integration,
              or API
            </li>
          </ul>
        </div>
      </div>

      <p className="text-center mt-4 opacity-70">
        Get started locally in one command:
      </p>

      <div className="mt-4 mb-20 flex gap-4 flex-col md:flex-row items-center justify-center">
        <div className="bg-white/10 backdrop-blur-md flex rounded text-sm text-slate-200 shadow-lg">
          <pre className=" pl-4 pr-2 py-2">
            <code className="bg-transparent text-slate-300">
              <span>npx</span> inngest-cli dev
            </code>
          </pre>
          <div className="rounded-r flex items-center justify-center pl-2 pr-2.5">
            <CopyBtn
              btnAction={handleCopyClick}
              copy="npx inngest-cli@latest dev"
            />
          </div>
        </div>
        <Link
          href="/docs/quick-start?ref=homepage-dev-tools"
          className="rounded-md px-3 py-1.5 text-sm bg-transparent transition-all text-white border border-slate-800 hover:border-slate-600 hover:bg-slate-500/10 whitespace-nowrap"
        >
          Get Started
        </Link>
      </div>
    </Container>
  );
};

const ProductionCopy = () => {
  const handleCopyClick = (copy) => {
    navigator.clipboard?.writeText(copy);
  };
  return (
    <Container className="py-8">
      <div className="grid lg:grid-cols-3 gap-20 pt-10 pb-20 px-10 opacity-80">
        <div>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            width={36}
            height={36}
            viewBox="0 0 24 24"
            className="my-6"
          >
            <path
              stroke="#fff"
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1}
              d="m9 6 3-3m0 0 3 3m-3-3v10m-5-3c-.932 0-1.398 0-1.765.152a2 2 0 0 0-1.083 1.083C4 11.602 4 12.068 4 13v4.8c0 1.12 0 1.68.218 2.108a2 2 0 0 0 .874.874c.427.218.987.218 2.105.218h9.607c1.118 0 1.677 0 2.104-.218.376-.192.682-.498.874-.874.218-.428.218-.987.218-2.105V13c0-.932 0-1.398-.152-1.765a2 2 0 0 0-1.083-1.083C18.398 10 17.932 10 17 10"
            />
          </svg>
          <h3
            className="
              text-xl font-semibold
              mb-4
              w-full
            "
          >
            No infra or CD changes
          </h3>
          <p>
            Deploy in your existing API, on your existing host, without spinning
            up new infra or provisioning new services — whether you use servers
            or serverless.
          </p>
          <ul className="my-6 leading-8 opacity-70 leading-snug">
            <li className="flex items-center mb-4">
              <Check size={14} className="mr-2" /> Hosted in your existing API
            </li>
            <li className="flex items-center mb-4">
              <Check size={14} className="mr-2" /> Serverless, servers, or edge
            </li>
            <li className="flex items-center mb-4">
              <Check size={14} className="mr-2" /> Zero additional infra or
              provisioning
            </li>
          </ul>
        </div>

        <div>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width={36}
            height={36}
            viewBox="0 0 24 24"
            fill="none"
            className="my-6"
          >
            <path
              stroke="#fff"
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1}
              d="m15 11-4 4-2-2m-5 3.8v-5.348c0-.534 0-.801.065-1.05a2 2 0 0 1 .28-.617c.145-.213.346-.39.748-.741l4.801-4.202c.746-.652 1.119-.978 1.538-1.102.37-.11.765-.11 1.135 0 .42.124.794.45 1.54 1.104l4.8 4.2c.403.352.603.528.748.74.127.19.222.398.28.618.065.249.065.516.065 1.05v5.352c0 1.118 0 1.677-.218 2.105a2 2 0 0 1-.875.873c-.427.218-.986.218-2.104.218H7.197c-1.118 0-1.678 0-2.105-.218a1.999 1.999 0 0 1-.874-.873C4 18.48 4 17.92 4 16.8Z"
            />
          </svg>
          <h3
            className="
              text-xl font-semibold
              mb-4
              w-full
            "
          >
            Modern SDLC
          </h3>
          <p>
            Hassle free development with preview environments, logging,
            one-click replay, and error reporting built in.
          </p>
          <ul className="my-6 leading-8 opacity-70 leading-snug">
            <li className="flex items-center mb-4">
              <Check size={14} className="mr-2" /> Preview & branch envs built
              in
            </li>
            <li className="flex items-center mb-4">
              <Check size={14} className="mr-2" /> Logging and error reporting
            </li>
            <li className="flex items-center mb-4">
              <Check size={14} className="mr-2" /> E2E encryption available
            </li>
          </ul>
        </div>

        <div>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width={36}
            height={36}
            viewBox="0 0 24 24"
            fill="none"
            className="my-6 ml-[-2px]"
          >
            <path
              stroke="#fff"
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1}
              d="m15 15 6 6m-11-4a7 7 0 1 1 0-14 7 7 0 0 1 0 14Z"
            />
          </svg>
          <h3
            className="
              text-xl font-semibold
              mb-4
              w-full
            "
          >
            End-to-end observability
          </h3>
          <p>
            Full insight without the fuss. Tag functions by user, account,
            context length, prompt rating, and see any data on any metric.
          </p>
          <ul className="my-6 leading-8 opacity-70 leading-snug">
            <li className="flex items-center mb-4">
              <Check size={14} className="mr-2" /> Real-time metrics
            </li>
            <li className="flex items-center mb-4">
              <Check size={14} className="mr-2" /> Function tagging with
              user-level cost basis
            </li>
            <li className="flex items-center mb-4">
              <Check size={14} className="mr-2" /> One click replay
            </li>
          </ul>
        </div>
      </div>
    </Container>
  );
};

const aiFlow = `
export const userWorkflow = inngest.createFunction(
  fnOptions, fnListener,
  async ({ event, step, ctx }) => {
    const similar = await step.run("query-vectordb", async () => {
      // Query a vectorDB for similar results given input
      const embedding = createEmedding(event.data.input);
      return await index.query({ vector: embedding, topK: 3 }).matches;
    });

    const response = await step.run("generate-llm-response", async () => {
      // Inject our prompt given similar search results and event.data.input
      const prompt = createAgentPrompt(similar, event.data.input);
      return await llm.createCompletion({
        model: "gpt-3.5-turbo",
        prompt,
      });
    });

    // Run as many chains and post-AI flows as you need, with retries, state
    // and orchestration managed for you.

    await step.run("save-to-db", async () => {
      // Connect to your standard DBs and APIs, as this is regular code that's
      // deployed in your existing stack
      await db.summaries.create({ requestID: event.data.requestID, response });
    });
  }
);
`;

const DevIcon = (props) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width={146}
    height={98}
    fill="none"
    {...props}
  >
    <g stroke="#fff" strokeWidth={1.249} opacity={0.3}>
      <path d="M26.144 25.334C19.298.527 44.658-1.998 62.959 5.492c10.678 3.853 17.69-8.128 30.919-3.347 13.228 4.782-2.63 19.842 17.611 27.572 20.24 7.73 10.359 42.314-7.332 42.712-17.69.399-12.83 6.216-30.52 19.523-17.69 13.308-47.493-5.18-40.082-24.702 7.41-19.524 0-15.061-7.411-41.916Z" />
      <path d="M64.218 44.64c-.79-2.362 2.91-6.174 5.885-4.547 2.724 1.49 5.498.485 7.331 1.149 4.825 1.748 3.05 2.674 5.207 4.75 2.625 2.527 3.877 9.143-2.157 9.143-4.962 0-3.478 2.41-7.194 3.479-3.716 1.068-7.93-.75-7.93-5.537 0-1.797.174-4.501-1.142-8.436Z" />
      <path d="M29.605 26.969C23.309 4.202 46.7 1.559 63.608 8.517c9.955 3.638 16.582-7.345 28.774-2.938 12.465 4.506-2.113 18.281 16.484 25.497 18.639 7.257 9.77 39.298-6.861 39.66-16.534.363-11.98 5.87-28.4 18.065-16.42 12.195-43.897-4.777-37.16-22.96 6.738-17.912.016-14.101-6.84-38.872Z" />
      <path d="M33.066 28.604C27.32 7.877 48.742 5.117 64.257 11.541c9.232 3.423 15.474-6.562 26.63-2.53 11.7 4.23-1.597 16.721 15.356 23.423 17.037 6.784 9.18 36.283-6.39 36.609-15.377.326-11.13 5.524-26.28 16.606s-40.3-4.374-34.236-21.218c6.063-16.3.031-13.141-6.272-35.829Z" />
      <path d="M36.526 30.238c-5.194-18.686 14.259-21.562 28.38-15.67 8.509 3.207 14.365-5.78 24.486-2.121 10.936 3.954-1.08 15.16 14.228 21.347 15.436 6.311 8.591 33.267-5.92 33.557-14.22.29-10.28 5.178-24.159 15.148-13.88 9.97-36.703-3.972-31.313-19.476 5.39-14.689.047-12.181-5.702-32.785Z" />
      <path d="M39.987 31.873c-4.644-16.646 12.84-19.639 25.568-14.28 7.786 2.993 13.257-4.996 22.341-1.712 10.173 3.678-.564 13.599 13.101 19.273 13.835 5.837 8.002 30.251-5.45 30.505-13.062.253-9.429 4.832-22.038 13.689s-33.107-3.57-28.39-17.733c4.715-13.078.063-11.222-5.132-29.742Z" />
      <path d="M43.448 33.59C39.355 18.987 54.87 15.877 66.204 20.7c7.063 2.78 12.149-4.212 20.197-1.302 9.409 3.402-.048 12.038 11.973 17.198 12.233 5.365 7.413 27.236-4.98 27.453-11.904.218-8.578 4.486-19.917 12.23-11.338 7.745-29.51-3.166-25.468-15.99 4.043-11.466.08-10.261-4.56-26.698Z" />
      <path d="M46.91 35.407c-3.544-12.565 10.002-15.792 19.943-11.5 6.34 2.563 11.04-3.43 18.053-.895 8.644 3.127.468 10.478 10.845 15.124 10.632 4.892 6.824 24.22-4.509 24.402-10.748.18-7.728 4.14-17.797 10.771-10.068 6.632-25.913-2.763-22.545-14.248 3.369-9.855.095-9.301-3.99-23.654Z" />
      <path d="M50.37 37.228c-2.992-10.525 8.584-13.869 17.132-10.11 5.617 2.35 9.932-2.647 15.909-.485 7.88 2.85.985 8.916 9.717 13.049 9.031 4.418 6.235 21.205-4.038 21.35-9.591.145-6.879 3.794-15.676 9.313-8.798 5.518-22.317-2.361-19.622-12.506 2.694-8.244.11-8.342-3.422-20.611Z" />
      <path d="M53.831 39.057c-2.441-8.484 7.166-11.945 14.32-8.72 4.894 2.135 8.824-1.863 13.765-.076 7.117 2.575 1.501 7.356 8.59 10.974 7.429 3.946 5.645 18.19-3.568 18.298-8.434.109-6.029 3.448-13.556 7.855-7.527 4.406-18.72-1.959-16.7-10.764 2.022-6.632.127-7.381-2.85-17.567Z" />
      <path d="M57.293 40.897C55.402 34.454 63.04 30.876 68.8 33.57c4.17 1.919 7.715-1.081 11.62.331 6.353 2.3 2.018 5.796 7.462 8.9 5.828 3.473 5.056 15.174-3.097 15.247-7.277.072-5.178 3.102-11.435 6.395-6.257 3.294-15.124-1.555-13.777-9.021 1.348-5.02.143-6.421-2.281-14.524Z" />
      <path d="M60.755 42.755c-1.34-4.403 4.328-8.098 8.696-5.938 3.448 1.705 6.607-.298 9.476.74 5.589 2.024 2.534 4.235 6.335 6.825 4.226 3 4.466 12.159-2.628 12.195-6.12.036-4.328 2.757-9.314 4.937-4.987 2.18-11.527-1.152-10.854-7.278.674-3.41.158-5.462-1.711-11.48Z" />
    </g>
    <rect width={146} height={29} y={33} fill="#050911" rx={4} />
    <path
      fill="#fff"
      d="M22.27 52v-8.531h1.974c.332.004.645.039.938.105.297.063.57.153.82.27.348.16.654.37.92.633a3 3 0 0 1 .644.896c.145.29.254.605.329.95.078.343.119.71.123 1.1v.628c0 .375-.038.73-.112 1.066a3.97 3.97 0 0 1-.31.926 3.45 3.45 0 0 1-1.254 1.406c-.282.176-.6.31-.955.404a4.591 4.591 0 0 1-1.143.147H22.27Zm1.101-7.64v6.755h.873c.313-.004.596-.043.85-.117.258-.074.486-.18.685-.316.211-.141.393-.313.545-.516.156-.207.28-.438.37-.691.078-.207.136-.43.175-.668.04-.242.06-.495.065-.756v-.639a4.765 4.765 0 0 0-.07-.762c-.04-.246-.1-.474-.182-.685a2.662 2.662 0 0 0-.428-.738 2.029 2.029 0 0 0-.65-.54 2.454 2.454 0 0 0-.616-.234 3.152 3.152 0 0 0-.744-.094h-.873Zm13.31 3.697H33.12v3.023h4.154V52h-5.238v-8.531h5.185v.925H33.12v2.743h3.562v.92ZM43.687 52l-2.696-8.531h1.16l1.876 6.369.123.416.129-.428 1.886-6.357h1.154L44.63 52h-.943Zm12.208-3.943h-3.562v3.023h4.154V52H51.25v-8.531h5.185v.925h-4.101v2.743h3.562v.92Zm6.139 3.023h4.172V52H60.95v-8.531h1.084v7.611Zm13.93-2.853a6.264 6.264 0 0 1-.081.925 4.939 4.939 0 0 1-.223.897c-.101.289-.232.558-.392.808-.157.25-.344.47-.563.657-.219.187-.47.336-.756.445a2.687 2.687 0 0 1-.949.158c-.352 0-.67-.052-.955-.158a2.58 2.58 0 0 1-.75-.445 2.964 2.964 0 0 1-.568-.657 4.012 4.012 0 0 1-.393-.814 4.958 4.958 0 0 1-.234-.896 6.141 6.141 0 0 1-.082-.92v-.973c.004-.305.029-.611.076-.92.05-.313.129-.613.234-.902.102-.29.23-.559.387-.809.16-.254.35-.476.568-.668.219-.187.469-.334.75-.44.285-.109.604-.163.955-.163.352 0 .67.054.955.164.285.105.538.252.756.44.219.187.407.407.563.661.16.25.293.52.398.809.102.289.176.59.223.902.05.313.078.621.082.926v.973Zm-1.071-.985a7.265 7.265 0 0 0-.041-.633 4.802 4.802 0 0 0-.118-.662 3.622 3.622 0 0 0-.228-.615 2.073 2.073 0 0 0-.352-.527 1.556 1.556 0 0 0-1.166-.487 1.542 1.542 0 0 0-1.16.492c-.14.153-.258.329-.351.528-.094.195-.168.4-.223.615-.059.219-.102.44-.129.662a7.238 7.238 0 0 0-.041.627v.985c.004.199.018.41.041.632.027.223.07.442.129.657.058.218.135.427.228.627.094.199.211.373.352.521.14.152.307.273.498.363.191.09.414.135.668.135a1.556 1.556 0 0 0 1.172-.498c.137-.148.25-.32.34-.516.094-.199.17-.408.228-.627a4.12 4.12 0 0 0 .112-.656c.023-.222.037-.435.04-.638v-.985Zm6.314 1.336V52h-1.084v-8.531h2.765c.383.008.744.068 1.084.181.344.114.645.278.903.493.257.214.46.48.609.797.152.316.229.68.229 1.09 0 .41-.077.773-.229 1.09a2.302 2.302 0 0 1-.61.79 2.725 2.725 0 0 1-.902.492 3.52 3.52 0 0 1-1.084.176h-1.681Zm0-.89h1.681c.25-.004.48-.044.692-.118.21-.078.394-.187.55-.328.157-.14.278-.31.364-.51.09-.203.135-.433.135-.691 0-.258-.045-.49-.135-.697a1.519 1.519 0 0 0-.358-.528 1.629 1.629 0 0 0-.556-.334 2.07 2.07 0 0 0-.692-.123h-1.681v3.328Zm9.613-4.22 1.388 4.266 1.5-4.265h1.348V52H94v-3.404l.088-3.563-1.576 4.594h-.621l-1.447-4.47.087 3.439V52h-1.054v-8.531h1.342Zm13.11 4.589h-3.562v3.023h4.154V52h-5.238v-8.531h5.186v.925h-4.102v2.743h3.562v.92ZM114.188 52h-1.102l-3.31-6.375-.018 6.375h-1.096v-8.531h1.102l3.311 6.363.017-6.363h1.096V52Zm10.017-7.605h-2.637V52h-1.054v-7.605h-2.637v-.926h6.328v.925Z"
    />
  </svg>
);

const ProdIcon = (props) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width={146}
    height={102}
    fill="none"
    {...props}
  >
    <g stroke="#fff" strokeWidth={1.199} opacity={0.3}>
      <path d="M63.554 6.897c-2.596-6.755-15.44-10.09-13.32 3.36.347 2.205.382 6.002 0 10.624-.351 4.235-7.21 6.552-9.316 12.023-3.303 8.58 9.044 14.56 7.08 19.071-2.574 5.908-10.454 8.47-14.812 11.703-4.103 3.043-6.717 7.925-7.72 12.944C23.704 85.428 26.9 94.656 35.7 95.115c4.406.23 10.216-1.738 17.513-7.056 3.263-2.378 6.14-4.337 8.696-5.922 6.581-4.078 4.812 5.568 12.637 5.922 4.459.201 9.115-8.126 12.64-3.684 2.507 3.158 5.384 6.99 9.492 10.93 7.214 6.92 16.234 7.939 17.51-1.936.602-4.662-.522-11.752-4.375-21.794-2.466-6.427 3.668-6.2 4.375-14.435.706-8.236-5.073-12.034-.889-15.442l.083-.068c3.189-2.598 6.59-5.369 8.448-10.69 1.6-4.584 1.447-9.124 0-12.612-1.549-3.734-4.58-6.263-8.531-6.353-3.965-.09-8.856 2.276-14.106 8.346-3.461 4.002-5.805-5.31-12.006-4.358-6.201.952-7.063 9.936-10.778 7.33-6.453-4.526-9.815-8.484-12.855-16.396Z" />
      <path d="M76.08 43.745c-.395-1.037-2.237-.765-2.33.36-.106 1.279.54 2.946.466 4.229-.144 2.477-1.02 3.182-2.761 4.487-.625.467-1.023 1.217-1.175 1.987-.268 1.352.218 2.769 1.557 2.839.67.035 1.555-.267 2.666-1.083.496-.365.934-.666 1.323-.91 1.694-1.143 3.27-.247 3.847.344.381.485.82 1.073 1.444 1.678 1.098 1.063 2.76.923 2.665-.297-.135-1.724-3.079-8.16 0-9.282 1.43-.522 1.383-1.703 1.163-2.238-.236-.573-.697-.962-1.298-.975-.604-.014-1.603.975-2.147 1.28-2.007 1.13-3.798 1.714-5.42-2.419Z" />
      <path d="M52.38 13.2c-1.88-12.245 9.857-9.193 12.238-3 2.914 7.57 6.136 11.12 12.19 15.136 3.404 2.357 4.224-5.847 9.902-6.738 5.678-.892 7.847 7.558 11.024 3.917 4.831-5.52 9.323-7.638 12.952-7.505 3.605.106 6.369 2.446 7.766 5.882 1.304 3.212 1.411 7.38-.109 11.575-1.985 4.985-4.852 8.088-7.763 10.625-3.802 3.156 1.421 6.642.729 14.14-.693 7.498-6.337 7.27-4.143 13.07 3.45 9.082 4.432 15.479 3.849 19.676-1.187 8.959-9.442 7.998-16.068 1.689-3.8-3.599-6.491-7.08-8.849-9.902-3.243-4.02-7.52 3.584-11.622 3.434-7.17-.285-5.606-9.036-11.634-5.31-2.367 1.457-5.02 3.246-8.018 5.407-6.762 4.833-12.093 6.495-16.077 6.164-8-.453-10.895-8.883-9.276-16.925.931-4.598 3.332-9.067 7.094-11.856 4.124-3.059 11.374-5.454 13.728-11.053 1.791-4.161-9.456-9.669-6.464-17.542 1.904-5.044 8.133-7.219 8.456-11.129.352-4.256.365-7.742.096-9.756Z" />
      <path d="M54.625 16.266c-1.64-10.999 8.974-8.24 11.135-2.63 2.783 7.205 5.86 10.332 11.506 13.825 3.09 2.1 3.865-5.295 9.012-6.123 5.146-.829 7.138 6.732 10.025 3.465 4.406-4.954 8.493-6.817 11.781-6.64 3.254.122 5.747 2.264 6.989 5.39 1.159 2.926 1.221 6.71-.219 10.5-2.093 4.586-4.377 8.009-6.982 10.457-3.415 2.893 1.244 6.056.567 12.791s-5.823 6.509-3.904 11.662c3.04 8.092 3.88 13.776 3.318 17.495-1.097 8.012-8.577 7.113-14.604 1.434-3.485-3.245-5.988-6.364-8.194-8.84-2.956-3.584-6.846 3.27-10.586 3.172-6.505-.215-5.148-8.043-10.614-4.681-2.173 1.325-4.598 2.937-7.33 4.876-6.215 4.33-11.059 5.682-14.616 5.251-7.189-.444-9.779-8.049-8.302-15.301.857-4.16 3.042-8.203 6.457-10.73 3.884-2.874 10.492-5.094 12.624-10.364 1.615-3.8-8.516-8.816-5.838-15.956 1.698-4.6 7.287-6.625 7.584-10.198.321-3.876.38-7.04.19-8.855Z" />
      <path d="M56.772 19.208c-1.401-9.792 8.105-7.317 10.05-2.27 2.657 6.865 5.594 9.582 10.84 12.566 2.78 1.851 3.513-4.763 8.136-5.531 4.623-.768 6.44 5.93 9.043 3.025 3.987-4.406 7.675-6.02 10.626-5.8 2.908.138 5.133 2.091 6.223 4.92 1.016 2.649 1.033 6.062-.328 9.463-2.205 4.202-3.909 7.955-6.212 10.323-3.034 2.64 1.068 5.492.406 11.49-.663 5.997-5.318 5.77-3.672 10.295 2.636 7.132 3.334 12.123 2.791 15.379-1.007 7.094-7.723 6.254-13.161 1.185-3.177-2.904-5.495-5.672-7.551-7.812-2.674-3.161-6.185 2.97-9.567 2.922-5.85-.146-4.698-7.078-9.611-4.069-1.983 1.198-4.184 2.64-6.652 4.362-5.68 3.844-10.043 4.891-13.178 4.359-6.389-.438-8.678-7.244-7.342-13.734.785-3.739 2.758-7.367 5.83-9.64 3.65-2.701 9.627-4.754 11.54-9.715 1.442-3.45-7.589-7.994-5.222-14.427 1.496-4.173 6.454-6.055 6.724-9.304.291-3.51.398-6.363.287-7.987Z" />
      <path d="M59.015 22.152c-1.16-8.587 7.222-6.395 8.947-1.91 2.526 6.524 5.318 8.831 10.157 11.304 2.464 1.603 3.153-4.23 7.244-4.938 4.092-.708 5.73 5.129 8.044 2.585 3.561-3.857 6.843-5.222 9.454-4.958 2.556.153 4.51 1.917 5.445 4.447.871 2.374.842 5.416-.436 8.426-2.313 3.82-3.436 7.905-5.432 10.192-2.647 2.389.891 4.928.244 10.187-.647 5.26-4.804 5.032-3.433 8.93 2.226 6.171 2.782 10.47 2.26 13.261-.917 6.177-6.856 5.395-11.696.936-2.862-2.563-4.991-4.979-6.896-6.782-2.386-2.739-5.51 2.668-8.53 2.672-5.184-.078-4.239-6.115-8.59-3.458a116.214 116.214 0 0 0-5.962 3.848c-5.134 3.358-9.01 4.1-11.718 3.466-5.576-.431-7.56-6.44-6.368-12.166.71-3.317 2.468-6.533 5.194-8.553 3.409-2.527 8.744-4.412 10.435-9.063 1.265-3.102-6.649-7.173-4.597-12.898 1.29-3.747 5.609-5.486 5.851-8.411.26-3.144.414-5.686.383-7.117Z" />
      <path d="M61.257 25.233c-.92-7.361 6.338-5.458 7.843-1.546 2.395 6.17 5.042 8.062 9.472 10.019 2.149 1.35 2.794-3.689 6.353-4.335 3.56-.646 5.02 4.316 7.044 2.14 3.136-3.3 6.012-4.414 8.281-4.107 2.204.17 3.887 1.74 4.668 3.966.726 2.092.652 4.756-.545 7.37-2.42 3.429-2.961 7.835-4.651 10.037-2.26 2.131.714 4.353.083 8.862-.632 4.51-4.29 4.283-3.194 7.544 1.817 5.197 2.23 8.793 1.729 11.114-.827 5.245-5.989 4.523-10.23.685-2.548-2.216-4.488-4.275-6.24-5.738-2.098-2.31-4.836 2.361-7.492 2.416-4.52-.01-3.781-5.137-7.57-2.838-1.596.941-3.341 2.038-5.272 3.325-4.587 2.865-7.976 3.299-10.257 2.565-4.764-.423-6.443-5.62-5.394-10.57.637-2.887 2.178-5.683 4.556-7.445 3.168-2.348 7.862-4.062 9.33-8.394 1.089-2.745-5.708-6.334-3.97-11.34 1.084-3.31 4.763-4.904 4.978-7.498.23-2.77.43-4.996.478-6.232Z" />
      <path d="M63.402 28.175c-.68-6.154 5.468-4.535 6.757-1.185 2.268 5.83 4.774 7.313 8.805 8.758 1.838 1.102 2.44-3.156 5.476-3.742 3.035-.585 4.32 3.514 6.059 1.699 2.716-2.75 5.193-3.615 7.125-3.265 1.857.186 3.272 1.567 3.901 3.495.582 1.816.463 4.109-.655 6.332-2.533 3.046-2.493 7.786-3.881 9.908-1.877 1.878.538 3.788-.079 7.56-.616 3.77-3.783 3.543-2.96 6.176 1.411 4.236 1.683 7.139 1.201 8.995-.737 4.327-5.134 3.663-8.785.435-2.239-1.874-3.993-3.582-5.596-4.708-1.816-1.886-4.173 2.06-6.472 2.166-3.862.06-3.33-4.172-6.565-2.225-1.405.813-2.926 1.74-4.593 2.81-4.05 2.379-6.958 2.507-8.816 1.671-3.962-.417-5.34-4.815-4.432-9.001.564-2.465 1.892-4.848 3.929-6.357 2.933-2.174 6.995-3.72 8.244-7.744.914-2.396-4.78-5.512-3.354-9.81.882-2.884 3.928-4.335 4.117-6.605.2-2.404.447-4.318.574-5.363Z" />
      <path d="M65.64 31.118c-.438-4.946 4.584-3.611 5.653-.824 2.135 5.49 4.495 6.563 8.115 7.497 1.523.852 2.081-2.623 4.584-3.149 2.503-.526 3.61 2.71 5.058 1.257 2.29-2.2 4.36-2.816 5.95-2.422 1.506.202 2.65 1.394 3.124 3.023.437 1.54.273 3.462-.763 5.295-2.638 2.663-2.018 7.739-3.1 9.781-1.49 1.626.362 3.224-.24 6.256-.6 3.032-3.267 2.804-2.72 4.808 1.003 3.274 1.133 5.483.672 6.873-.647 3.409-4.266 2.803-7.318.186-1.923-1.532-3.488-2.888-4.937-3.677-1.528-1.463-3.498 1.759-5.434 1.916-3.196.13-2.87-3.206-5.543-1.612-1.21.686-2.503 1.442-3.901 2.296-3.503 1.892-5.923 1.713-7.353.775-3.149-.41-4.222-4.01-3.458-7.432.49-2.043 1.602-4.012 3.291-5.267 2.69-2.002 6.11-3.38 7.136-7.095.738-2.048-3.839-4.69-2.727-8.28.676-2.457 3.082-3.766 3.243-5.712.17-2.038.464-3.64.669-4.493Z" />
      <path d="M67.817 34.06c-.2-3.737 3.724-2.686 4.579-.463 2.015 5.155 4.242 5.816 7.472 6.237 1.215.602 1.733-2.09 3.717-2.556 1.984-.466 2.918 1.905 4.085.816 1.876-1.65 3.55-2.017 4.808-1.578 1.161.218 2.04 1.22 2.362 2.551.294 1.264.084 2.814-.876 4.256-2.76 2.281-1.554 7.696-2.335 9.66-1.11 1.374.186 2.659-.402 4.951-.588 2.293-2.77 2.064-2.496 3.438.599 2.31.587 3.823.144 4.748-.56 2.487-3.421 1.94-5.89-.065-1.619-1.19-3.003-2.194-4.306-2.644-1.25-1.04-2.843 1.456-4.426 1.665-2.546.199-2.426-2.238-4.55-.997l-.06.032a313.619 313.619 0 0 0-3.172 1.749c-2.975 1.404-4.92.917-5.93-.124-2.353-.404-3.127-3.204-2.502-5.86.418-1.62 1.32-3.177 2.67-4.178 2.465-1.829 5.26-3.04 6.07-6.448.564-1.698-2.92-3.868-2.117-6.749.475-2.03 2.253-3.197 2.388-4.818.139-1.671.482-2.962.767-3.623Z" />
      <path d="M70.051 37.145c.041-2.51 2.835-1.748 3.47-.099 1.88 4.796 3.957 5.041 6.773 4.948.898.35 1.37-1.548 2.82-1.951 1.45-.404 2.204 1.092 3.08.37 1.448-1.093 2.715-1.208 3.629-.726.809.233 1.415 1.042 1.582 2.068.15.981-.106 2.153-.982 3.198-2.86 1.888-1.078 7.618-1.552 9.494-.722 1.115.01 2.082-.562 3.625-.571 1.542-2.252 1.314-2.252 2.052.19 1.336.035 2.148-.386 2.602-.468 1.557-2.55 1.07-4.416-.315-1.302-.843-2.495-1.49-3.644-1.6-.96-.61-2.165 1.149-3.382 1.408-1.877.267-1.964-1.26-3.524-.378-.828.429-1.67.84-2.537 1.258-2.423.91-3.878.118-4.46-1.022-1.539-.396-2.007-2.383-1.526-4.263.343-1.19 1.028-2.326 2.03-3.07 2.218-1.647 4.367-2.685 4.953-5.771.388-1.341-1.975-3.028-1.488-5.188.27-1.593 1.407-2.613 1.514-3.903.107-1.297.497-2.271.86-2.737Z" />
      <path d="M72.318 40.092c.28-1.294 1.954-.819 2.37.263 1.747 4.446 3.678 4.28 6.086 3.673.584.099 1.012-1.01 1.932-1.353.92-.342 1.496.285 2.084-.073 1.023-.54 1.885-.404 2.46.12.458.25.794.867.808 1.592.005.702-.295 1.499-1.09 2.15-2.964 1.5-.605 7.556-.774 9.348-.337.86-.167 1.512-.722 2.31-.555.798-1.739.57-2.013.675-.218.367-.513.481-.914.468-.377.631-1.684.203-2.955-.565-.988-.498-1.991-.79-2.987-.562-.673-.185-1.493.843-2.349 1.154-1.214.336-1.506-.289-2.506.238-.635.3-1.249.539-1.85.739-1.877.42-2.846-.679-3.003-1.918-.73-.39-.895-1.571-.557-2.68.27-.765.739-1.483 1.395-1.97 1.977-1.471 3.486-2.34 3.85-5.11.213-.988-1.038-2.197-.864-3.642.065-1.162.565-2.037.645-2.999.077-.927.512-1.587.954-1.858Z" />
    </g>
    <rect width={146} height={29} y={41} fill="#050911" rx={4} />
    <path
      fill="#fff"
      d="M28.368 56.578V60h-1.084v-8.531h2.766c.382.008.744.068 1.084.181.343.114.644.278.902.493.258.214.46.48.61.797.152.316.228.68.228 1.09 0 .41-.076.773-.229 1.09a2.3 2.3 0 0 1-.61.79 2.725 2.725 0 0 1-.901.492 3.52 3.52 0 0 1-1.084.176h-1.682Zm0-.89h1.682c.25-.004.48-.044.691-.118.211-.078.395-.187.55-.328.157-.14.278-.31.364-.51.09-.203.135-.433.135-.691 0-.258-.045-.49-.135-.697a1.52 1.52 0 0 0-.357-.528 1.629 1.629 0 0 0-.557-.334 2.07 2.07 0 0 0-.691-.123h-1.682v3.328Zm11.248.831H37.91V60h-1.078v-8.531h2.502c.399.008.774.064 1.125.17.352.105.66.263.926.474.262.211.467.477.615.797.153.317.229.69.229 1.12 0 .277-.041.53-.123.761a2.23 2.23 0 0 1-.328.627c-.14.188-.309.354-.504.498-.196.145-.41.268-.645.37l1.81 3.644-.005.07h-1.143l-1.675-3.48Zm-1.706-.89h1.454c.242-.004.47-.041.685-.111.215-.075.404-.18.569-.317a1.5 1.5 0 0 0 .38-.498c.094-.2.141-.428.141-.685 0-.274-.045-.512-.135-.715a1.452 1.452 0 0 0-.375-.522 1.628 1.628 0 0 0-.574-.31 2.467 2.467 0 0 0-.72-.112H37.91v3.27Zm14.037.598a6.264 6.264 0 0 1-.082.925 4.93 4.93 0 0 1-.222.897c-.102.289-.233.558-.393.808-.156.25-.344.47-.563.657-.218.187-.47.336-.755.445a2.687 2.687 0 0 1-.95.158c-.351 0-.67-.052-.955-.158a2.58 2.58 0 0 1-.75-.445 2.964 2.964 0 0 1-.568-.657 4.018 4.018 0 0 1-.393-.814 4.958 4.958 0 0 1-.234-.896 6.141 6.141 0 0 1-.082-.92v-.973c.004-.305.03-.611.076-.92.051-.313.13-.613.234-.902a4.06 4.06 0 0 1 .387-.809c.16-.254.35-.476.569-.668.218-.187.468-.334.75-.44.285-.109.603-.163.955-.163.351 0 .67.054.955.164.285.105.537.252.756.44.218.187.406.407.562.661.16.25.293.52.399.809.101.289.175.59.222.902.051.313.078.621.082.926v.973Zm-1.072-.985a7.265 7.265 0 0 0-.041-.633 4.802 4.802 0 0 0-.117-.662 3.622 3.622 0 0 0-.229-.615 2.073 2.073 0 0 0-.351-.527 1.557 1.557 0 0 0-1.166-.487 1.542 1.542 0 0 0-1.16.492c-.141.153-.258.329-.352.528-.094.195-.168.4-.223.615-.058.219-.101.44-.129.662a7.238 7.238 0 0 0-.04.627v.985c.003.199.017.41.04.632.028.223.07.442.13.657.058.218.134.427.228.627.094.199.21.373.351.521.141.152.307.273.498.363.192.09.414.135.668.135a1.556 1.556 0 0 0 1.172-.498c.137-.148.25-.32.34-.516.094-.199.17-.408.229-.627a4.12 4.12 0 0 0 .111-.656c.023-.222.037-.435.041-.638v-.985ZM55.895 60v-8.531h1.974c.332.004.644.039.937.105.297.063.57.153.82.27.348.16.655.37.92.633.27.257.485.556.645.896.145.29.254.605.328.95.078.343.12.71.123 1.1v.628c0 .375-.037.73-.111 1.066-.07.336-.174.645-.31.926a3.45 3.45 0 0 1-1.254 1.406c-.282.176-.6.31-.956.404A4.591 4.591 0 0 1 57.87 60h-1.975Zm1.1-7.64v6.755h.874c.312-.004.596-.043.85-.117.257-.074.486-.18.685-.316.21-.141.392-.313.545-.516.156-.207.28-.438.369-.691.078-.207.137-.43.176-.668a5.24 5.24 0 0 0 .064-.756v-.639a4.753 4.753 0 0 0-.07-.762 3.33 3.33 0 0 0-.182-.685 2.665 2.665 0 0 0-.427-.738 2.029 2.029 0 0 0-.65-.54 2.452 2.452 0 0 0-.616-.234 3.151 3.151 0 0 0-.744-.094h-.873Zm13.973-.891.012 5.777c0 .398-.07.772-.211 1.12-.14.347-.334.65-.58.907-.246.262-.54.47-.88.621-.34.149-.712.223-1.118.223-.414 0-.791-.074-1.131-.222a2.582 2.582 0 0 1-.873-.616 2.815 2.815 0 0 1-.569-.908 3.162 3.162 0 0 1-.21-1.125l.011-5.777h1.031l.024 5.777c.004.254.043.498.117.732.078.235.19.442.334.622.14.18.316.324.527.433.215.11.461.164.739.164.277 0 .521-.053.732-.158.211-.11.389-.256.533-.44.14-.18.248-.386.322-.62.075-.235.116-.479.124-.733l.017-5.777h1.049Zm9.777 5.965c-.05.394-.152.757-.305 1.09a2.8 2.8 0 0 1-.592.843 2.629 2.629 0 0 1-.873.557 3.054 3.054 0 0 1-1.113.193c-.352 0-.672-.049-.96-.146a2.768 2.768 0 0 1-1.348-1.037 3.755 3.755 0 0 1-.399-.78 4.695 4.695 0 0 1-.246-.885 6.237 6.237 0 0 1-.082-.937v-1.19c.004-.316.031-.628.082-.937.055-.308.137-.603.246-.885.105-.28.238-.54.398-.779.165-.242.36-.451.587-.627.222-.176.476-.312.761-.41a2.85 2.85 0 0 1 .961-.152c.426 0 .807.066 1.143.199.336.129.625.312.867.55.242.243.435.532.58.868.148.336.246.707.293 1.113H79.66a3.193 3.193 0 0 0-.182-.72 1.986 1.986 0 0 0-.345-.587 1.493 1.493 0 0 0-.534-.392 1.717 1.717 0 0 0-.738-.147c-.258 0-.486.043-.685.13a1.51 1.51 0 0 0-.504.34 1.974 1.974 0 0 0-.364.503c-.093.191-.17.394-.228.61a4.605 4.605 0 0 0-.129.661c-.023.223-.035.44-.035.65v1.202a4.633 4.633 0 0 0 .164 1.313c.058.218.135.423.228.615.094.191.213.36.358.504.144.148.314.265.51.351.195.082.423.123.685.123.285 0 .531-.045.738-.135.211-.09.39-.216.534-.38.144-.16.26-.35.345-.569.086-.222.147-.463.182-.72h1.084Zm9.835-5.04h-2.636V60h-1.055v-7.605h-2.637v-.926h6.329v.925Zm3.854-.925h5.156v.943h-2.045v6.65h2.045V60h-5.156v-.938h1.998v-6.65h-1.998v-.943Zm15.155 4.758a6.118 6.118 0 0 1-.082.925 4.905 4.905 0 0 1-.222.897c-.102.289-.233.558-.393.808-.156.25-.344.47-.562.657a2.564 2.564 0 0 1-.756.445 2.687 2.687 0 0 1-.949.158c-.352 0-.67-.052-.955-.158a2.559 2.559 0 0 1-.75-.445 2.957 2.957 0 0 1-.569-.657 3.995 3.995 0 0 1-.392-.814 4.935 4.935 0 0 1-.235-.896 6.128 6.128 0 0 1-.082-.92v-.973c.004-.305.029-.611.076-.92.051-.313.129-.613.235-.902a4.04 4.04 0 0 1 .386-.809c.161-.254.35-.476.569-.668.219-.187.469-.334.75-.44.285-.109.603-.163.955-.163.351 0 .67.054.955.164.285.105.537.252.756.44.219.187.406.407.562.661.16.25.293.52.399.809.101.289.176.59.222.902.051.313.079.621.082.926v.973Zm-1.072-.985a7.074 7.074 0 0 0-.041-.633 4.685 4.685 0 0 0-.117-.662 3.632 3.632 0 0 0-.229-.615 2.04 2.04 0 0 0-.351-.527 1.565 1.565 0 0 0-1.166-.487 1.54 1.54 0 0 0-1.16.492 2.071 2.071 0 0 0-.352.528 3.23 3.23 0 0 0-.222.615c-.059.219-.102.44-.129.662a6.688 6.688 0 0 0-.041.627v.985c.003.199.017.41.041.632.027.223.07.442.129.657.058.218.134.427.228.627.094.199.211.373.352.521.14.152.306.273.498.363.191.09.414.135.668.135.254 0 .476-.045.668-.135.195-.09.363-.21.503-.363.137-.148.25-.32.34-.516.094-.199.17-.408.229-.627a4.02 4.02 0 0 0 .111-.656c.024-.222.037-.435.041-.638v-.985ZM118.991 60h-1.101l-3.311-6.375-.017 6.375h-1.096v-8.531h1.102l3.31 6.363.018-6.363h1.095V60Z"
    />
  </svg>
);

const StoreLogos = (props) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    xmlnsXlink="http://www.w3.org/1999/xlink"
    width={401}
    height={75}
    fill="none"
    {...props}
  >
    <g opacity={0.8}>
      <g clipPath="url(#a)">
        <path
          fill="#fff"
          d="M23.606 29.1h5.418c3.92 0 4.914 2.383 4.914 4.325 0 1.943-1.015 4.326-4.914 4.326h-3.343v6.814h-2.075V29.1Zm2.075 6.813h2.735c1.646 0 3.291-.388 3.291-2.487 0-2.1-1.645-2.488-3.29-2.488H25.68v4.975Zm10.97-6.888c.28 0 .555.084.789.24a1.44 1.44 0 0 1 .616 1.465 1.447 1.447 0 0 1-.377.738 1.428 1.428 0 0 1-1.54.346 1.431 1.431 0 0 1-.653-.508 1.45 1.45 0 0 1-.17-1.353 1.452 1.452 0 0 1 .776-.811 1.43 1.43 0 0 1 .559-.117Zm-.956 5.185h1.927v10.355h-1.927V34.21Zm4.158 0h1.949v1.599h.044a3.638 3.638 0 0 1 1.425-1.42 3.601 3.601 0 0 1 1.955-.44c1.994 0 3.706 1.202 3.706 3.952v6.664h-1.927v-6.112c0-1.95-1.104-2.667-2.342-2.667-1.623 0-2.861 1.046-2.861 3.452v5.327h-1.95V34.21Zm12.651 5.902c0 1.808 1.668 2.988 3.469 2.988a3.522 3.522 0 0 0 1.592-.433c.49-.269.909-.65 1.224-1.113l1.483 1.135a5.382 5.382 0 0 1-2.02 1.64 5.336 5.336 0 0 1-2.546.497c-3.254 0-5.284-2.353-5.284-5.439a5.269 5.269 0 0 1 .342-2.088 5.236 5.236 0 0 1 1.135-1.781 5.186 5.186 0 0 1 1.743-1.181 5.152 5.152 0 0 1 2.064-.389c3.617 0 5.003 2.795 5.003 5.462v.702h-8.205Zm6.167-1.576c-.045-1.726-1-2.989-2.965-2.989-.809-.005-1.59.301-2.182.857a3.208 3.208 0 0 0-1.005 2.132h6.152Zm10.888-1.57a3.224 3.224 0 0 0-2.557-1.158c-2.12 0-3.21 1.727-3.21 3.669a3.35 3.35 0 0 0 .903 2.486 3.299 3.299 0 0 0 2.418 1.033 3.034 3.034 0 0 0 2.49-1.158l1.386 1.382a5.053 5.053 0 0 1-1.777 1.224c-.673.274-1.396.4-2.121.367a5.084 5.084 0 0 1-2.093-.336 5.117 5.117 0 0 1-1.779-1.162 5.2 5.2 0 0 1-1.502-3.896 5.267 5.267 0 0 1 1.5-3.918 5.183 5.183 0 0 1 1.776-1.18 5.151 5.151 0 0 1 2.098-.363 5.238 5.238 0 0 1 2.142.399 5.275 5.275 0 0 1 1.8 1.237l-1.474 1.375ZM76.755 33.948a5.36 5.36 0 0 1 3.795 1.653 5.478 5.478 0 0 1-.126 7.697 5.39 5.39 0 0 1-1.772 1.15c-.66.261-1.365.39-2.075.377a5.375 5.375 0 0 1-3.782-1.66 5.462 5.462 0 0 1-1.514-3.868 5.46 5.46 0 0 1 1.64-3.816 5.374 5.374 0 0 1 3.834-1.533Zm0 9.048c2.076 0 3.38-1.495 3.38-3.609s-1.304-3.6-3.38-3.6c-2.075 0-3.387 1.493-3.387 3.6s1.305 3.609 3.387 3.609Zm7.041-8.786h1.95v1.599c.334-.6.83-1.092 1.428-1.421a3.608 3.608 0 0 1 1.959-.44c1.994 0 3.706 1.203 3.706 3.953v6.663h-1.95v-6.11c0-1.95-1.104-2.668-2.334-2.668-1.63 0-2.861 1.046-2.861 3.452v5.327h-1.898V34.21Zm12.674 5.902c0 1.808 1.668 2.988 3.47 2.988a3.545 3.545 0 0 0 1.59-.436 3.577 3.577 0 0 0 1.226-1.11l1.482 1.135a5.379 5.379 0 0 1-2.016 1.626 5.328 5.328 0 0 1-2.535.489c-3.246 0-5.284-2.354-5.284-5.44a5.267 5.267 0 0 1 1.485-3.877 5.185 5.185 0 0 1 1.75-1.18 5.152 5.152 0 0 1 2.071-.381c3.625 0 5.011 2.794 5.011 5.461v.703l-8.25.022Zm6.152-1.576c-.044-1.726-.993-2.989-2.965-2.989a3.15 3.15 0 0 0-2.184.854 3.2 3.2 0 0 0-1.002 2.135h6.151Z"
        />
        <path
          stroke="#fff"
          strokeLinecap="square"
          strokeWidth={1.77}
          d="m10.806 28.218.519-2.906"
        />
        <path
          stroke="#fff"
          strokeLinecap="square"
          strokeLinejoin="round"
          strokeWidth={1.77}
          d="m13.193 27.187-1.809-2.226-2.46 1.45"
        />
        <path
          stroke="#fff"
          strokeLinecap="square"
          strokeWidth={1.77}
          d="m8.642 40.62.504-2.906"
        />
        <path
          stroke="#fff"
          strokeLinecap="square"
          strokeLinejoin="round"
          strokeWidth={1.77}
          d="m11.029 39.581-1.824-2.219-2.453 1.457"
        />
        <path
          stroke="#fff"
          strokeLinecap="square"
          strokeWidth={1.77}
          d="m9.687 34.628.504-2.906"
        />
        <path
          stroke="#fff"
          strokeLinecap="square"
          strokeLinejoin="round"
          strokeWidth={1.77}
          d="m12.074 33.59-1.816-2.212-2.453 1.45"
        />
        <path
          fill="#fff"
          d="M7.982 45.58c.668 0 1.208-.545 1.208-1.217 0-.673-.54-1.218-1.208-1.218-.667 0-1.208.545-1.208 1.218 0 .672.541 1.218 1.208 1.218Z"
        />
        <path
          stroke="#fff"
          strokeLinecap="square"
          strokeWidth={1.68}
          d="M4.558 40.366 2.342 41.92"
        />
        <path
          stroke="#fff"
          strokeLinecap="square"
          strokeLinejoin="round"
          strokeWidth={1.68}
          d="m4.692 42.883-2.617-.776.193-2.742"
        />
        <path
          stroke="#fff"
          strokeLinecap="square"
          strokeWidth={1.68}
          d="m12.607 41.822 1.542 2.242"
        />
        <path
          stroke="#fff"
          strokeLinecap="square"
          strokeLinejoin="round"
          strokeWidth={1.68}
          d="m11.614 44.139 2.72.186.771-2.622"
        />
        <path
          stroke="#fff"
          strokeLinecap="square"
          strokeWidth={1.72}
          d="m15.135 37.273 2.712.493"
        />
        <path
          stroke="#fff"
          strokeLinecap="square"
          strokeLinejoin="round"
          strokeWidth={1.72}
          d="m16.069 39.67 2.105-1.844-1.327-2.458"
        />
        <path
          stroke="#fff"
          strokeLinecap="square"
          strokeWidth={1.72}
          d="m14.342 31.849 2.416-1.345"
        />
        <path
          stroke="#fff"
          strokeLinecap="square"
          strokeLinejoin="round"
          strokeWidth={1.72}
          d="m14.475 29.271 2.572 1.076-.482 2.757"
        />
        <path
          stroke="#fff"
          strokeLinecap="square"
          strokeWidth={1.72}
          d="m3.684 35.24-2.72-.477"
        />
        <path
          stroke="#fff"
          strokeLinecap="square"
          strokeLinejoin="round"
          strokeWidth={1.72}
          d="M1.986 37.168.637 34.71l2.083-1.852"
        />
        <path
          stroke="#fff"
          strokeLinecap="square"
          strokeWidth={1.72}
          d="m6.263 30.407-1.816-2.092"
        />
        <path
          stroke="#fff"
          strokeLinecap="square"
          strokeLinejoin="round"
          strokeWidth={1.72}
          d="m7.011 27.95-2.779.119-.49 2.757"
        />
      </g>
      <g fill="#fff" clipPath="url(#b)">
        <path d="M325.126 27.904c-4.455-4.363-11.678-4.363-16.133 0l-7.282 7.13a1.06 1.06 0 0 0 0 1.524l7.282 7.131c4.455 4.363 11.678 4.363 16.133.007a10.989 10.989 0 0 0 0-15.792Zm-1.721 13.697c-3.272 3.204-8.58 3.204-11.851 0l-5.359-5.243a.784.784 0 0 1 0-1.123L311.546 30c3.272-3.205 8.58-3.205 11.851 0a8.077 8.077 0 0 1 .008 11.6ZM334.667 35.042l-3.206-3.197c-.194-.193-.517-.014-.452.25a16.85 16.85 0 0 1 0 7.424c-.058.265.265.437.452.25l3.206-3.197a1.082 1.082 0 0 0 0-1.53Z" />
        <path d="M317.515 41.408c3.102 0 5.617-2.508 5.617-5.6 0-3.094-2.515-5.601-5.617-5.601s-5.617 2.507-5.617 5.6c0 3.093 2.515 5.6 5.617 5.6ZM341.383 34.82c0-.596.103-1.109.309-1.54a3.24 3.24 0 0 1 .865-1.11 3.564 3.564 0 0 1 1.298-.677 5.877 5.877 0 0 1 1.628-.226c.721 0 1.359.123 1.936.37.556.246 1.03.575 1.38 1.006.35-.431.804-.76 1.36-1.006.556-.247 1.195-.37 1.916-.37.576 0 1.112.082 1.606.226.495.144.927.37 1.298.678.371.308.659.677.865 1.109.206.431.309.944.309 1.54v5.874h-2.904v-5.052c0-.617-.124-1.048-.371-1.315-.247-.267-.597-.39-1.071-.39s-.845.164-1.154.472c-.288.308-.432.801-.432 1.5v4.806h-2.905v-4.806c0-.678-.144-1.171-.433-1.5-.288-.308-.659-.472-1.133-.472-.453 0-.824.123-1.071.39-.268.267-.391.699-.391 1.315v5.052h-2.905V34.82ZM358.564 29.07c0-.453.144-.843.453-1.151.309-.308.68-.473 1.153-.473.454 0 .845.165 1.154.473.309.308.474.698.474 1.15 0 .452-.165.842-.474 1.15-.309.308-.7.452-1.154.452-.453 0-.844-.144-1.153-.452-.289-.328-.453-.698-.453-1.15Zm.164 3.737c0-.472.145-.8.412-.986.268-.185.639-.287 1.092-.287.33 0 .618.04.886.123.268.082.433.123.515.144v8.914h-2.905v-7.908ZM364.311 28.72c0-.472.144-.801.412-.986.268-.185.638-.288 1.092-.288.329 0 .618.042.885.124.268.082.433.123.516.143v12.981h-2.905V28.72ZM372.53 40.694a46.912 46.912 0 0 1-1.092-2.033c-.35-.699-.679-1.356-.968-1.952a67.88 67.88 0 0 1-.721-1.54 17.013 17.013 0 0 0-.371-.822 16.593 16.593 0 0 1-.309-.8 2.146 2.146 0 0 1-.165-.802c0-.328.124-.616.371-.862.248-.226.598-.35 1.092-.35.392 0 .7.042.948.124.247.082.37.123.412.144.206.595.412 1.19.659 1.787.227.595.453 1.15.659 1.663.206.514.392.966.577 1.356.165.39.288.657.371.821.082-.164.185-.431.35-.8.165-.37.329-.802.515-1.254.185-.451.371-.924.556-1.376.186-.451.351-.862.474-1.17.083-.185.145-.35.248-.514.082-.164.205-.287.329-.41.123-.124.288-.206.453-.268.185-.061.412-.082.659-.082.247 0 .453.02.659.082.186.041.371.103.516.165.144.061.268.123.35.164.082.062.144.082.165.103-.042.226-.165.534-.35.986-.186.43-.392.944-.639 1.499-.248.554-.515 1.15-.804 1.766-.288.616-.577 1.212-.844 1.787a37.138 37.138 0 0 1-.783 1.561c-.247.452-.433.822-.556 1.068h-2.761v-.041ZM383.675 36.422c0 1.253.556 1.869 1.689 1.869s1.69-.616 1.69-1.87v-3.614c0-.472.144-.801.412-.986.268-.185.638-.287 1.091-.287.33 0 .618.04.886.123.268.082.433.123.515.144v5.155c0 .678-.124 1.253-.35 1.746a3.362 3.362 0 0 1-.969 1.232 4.073 4.073 0 0 1-1.462.74 6.439 6.439 0 0 1-1.813.246 6.344 6.344 0 0 1-1.813-.247 4.262 4.262 0 0 1-1.462-.739c-.412-.329-.721-.74-.968-1.232-.227-.493-.351-1.089-.351-1.746v-4.149c0-.472.145-.801.413-.986.267-.185.638-.287 1.091-.287.33 0 .618.04.886.123.268.082.433.123.515.144v4.62ZM392.904 37.9c.103.083.247.165.474.268.227.103.494.205.803.287.309.083.659.165 1.051.226.391.062.803.103 1.236.103.494 0 .865-.041 1.112-.144.247-.082.371-.246.371-.472 0-.247-.103-.41-.309-.493-.206-.082-.536-.164-.989-.226l-.947-.103a9.353 9.353 0 0 1-1.36-.267 3.923 3.923 0 0 1-1.153-.513 2.415 2.415 0 0 1-.783-.863c-.186-.349-.289-.78-.289-1.314 0-.452.082-.863.247-1.233.165-.37.412-.698.763-.985.35-.288.803-.514 1.338-.658.536-.164 1.195-.246 1.937-.246.762 0 1.401.04 1.957.144.556.102 1.01.246 1.421.451.227.124.412.268.556.432.145.164.206.39.206.637 0 .184-.041.37-.123.513-.083.144-.165.287-.268.39a2.71 2.71 0 0 1-.288.247c-.083.061-.144.102-.165.102-.041-.04-.144-.102-.288-.205-.165-.103-.371-.185-.659-.288a13.105 13.105 0 0 0-.969-.246c-.371-.082-.803-.103-1.297-.103-.536 0-.928.062-1.134.185-.206.123-.309.288-.309.493 0 .206.103.35.289.431.185.082.474.165.865.226l1.648.226c.412.062.803.144 1.174.267.371.123.721.288 1.009.493.289.226.536.493.701.822.165.328.268.74.268 1.232 0 1.006-.392 1.787-1.154 2.362-.762.554-1.833.842-3.193.842-.7 0-1.339-.041-1.895-.144a13.472 13.472 0 0 1-1.421-.328 6.698 6.698 0 0 1-.969-.39c-.247-.144-.412-.247-.515-.309l1.051-1.848Z" />
      </g>
      <path fill="url(#c)" d="M144.735.465h116.639v74.07H144.735z" />
    </g>
    <defs>
      <clipPath id="a">
        <path fill="#fff" d="M0 24.303h104.72v21.285H0z" />
      </clipPath>
      <clipPath id="b">
        <path fill="#fff" d="M301.389 24.303h99.612V47.29h-99.612z" />
      </clipPath>
      <pattern
        id="c"
        width={1}
        height={1}
        patternContentUnits="objectBoundingBox"
      >
        <use xlinkHref="#d" transform="matrix(.0005 0 0 .00079 -.002 0)" />
      </pattern>
      <image
        xlinkHref="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAB9AAAATyCAYAAADhtHrZAAAACXBIWXMAAAsTAAALEwEAmpwYAAAgAElEQVR4nOzdCZRtZ1UncDKahIQkECNBDIJBBpFJ0UZQMGA6NGAzKlEUmkEhgDQRFAUFQRQFRRAcUJnUtAgqM9gCymhoQGYkyhQQEghjEiDJ833/Xt/zO7W+d+rcuvdW1Xuv6tXvt9ZdNd06dzpn37vO/vbeV7kKAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA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AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHzboeX/AMgQzOYCIyWsAAAAAElFTkSuQmCC"
        id="d"
        width={2000}
        height={1266}
      />
    </defs>
  </svg>
);

const ProviderLogos = (props) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width={343}
    height={37}
    fill="none"
    {...props}
  >
    <g opacity={0.8}>
      <path
        fill="#fff"
        d="M192.013 19.458c0 3.807 2.46 6.462 5.868 6.462 3.409 0 5.868-2.655 5.868-6.462s-2.459-6.462-5.868-6.462c-3.408 0-5.868 2.654-5.868 6.462Zm9.488 0c0 2.724-1.494 4.488-3.62 4.488-2.126 0-3.619-1.764-3.619-4.488 0-2.725 1.493-4.489 3.619-4.489 2.126 0 3.62 1.764 3.62 4.489ZM210.083 25.92c2.582 0 4.058-2.166 4.058-4.768s-1.476-4.768-4.058-4.768c-1.195 0-2.074.472-2.653 1.153v-.978h-2.109V28.87h2.109v-4.104c.579.681 1.458 1.153 2.653 1.153Zm-2.706-5.03c0-1.73.984-2.672 2.284-2.672 1.528 0 2.354 1.187 2.354 2.934 0 1.746-.826 2.934-2.354 2.934-1.3 0-2.284-.96-2.284-2.655v-.541ZM219.847 25.92c1.845 0 3.303-.96 3.953-2.568l-1.81-.68c-.281.942-1.107 1.466-2.143 1.466-1.353 0-2.302-.96-2.46-2.532h6.466v-.699c0-2.515-1.423-4.523-4.094-4.523-2.671 0-4.392 2.078-4.392 4.768 0 2.83 1.844 4.768 4.48 4.768Zm-.106-7.772c1.336 0 1.968.873 1.986 1.886h-4.235c.317-1.24 1.16-1.886 2.249-1.886ZM225.435 25.728h2.108v-5.38c0-1.31.966-2.008 1.915-2.008 1.16 0 1.617.82 1.617 1.956v5.432h2.108v-6.043c0-1.974-1.16-3.301-3.092-3.301-1.195 0-2.021.541-2.548 1.153v-.978h-2.108v9.169ZM239.358 13.188l-4.779 12.54h2.231l1.072-2.865h5.447l1.089 2.864h2.266l-4.779-12.539h-2.547Zm1.23 2.48 2.003 5.24h-3.971l1.968-5.24ZM250.304 13.224h-2.249v12.54h2.249v-12.54ZM186.861 17.794a5.772 5.772 0 0 0-.502-4.765 5.916 5.916 0 0 0-6.357-2.815 5.852 5.852 0 0 0-4.402-1.951 5.9 5.9 0 0 0-5.63 4.062 5.836 5.836 0 0 0-3.903 2.814 5.842 5.842 0 0 0 .726 6.88 5.772 5.772 0 0 0 .502 4.765 5.915 5.915 0 0 0 6.357 2.814 5.85 5.85 0 0 0 4.402 1.95 5.9 5.9 0 0 0 5.632-4.064 5.836 5.836 0 0 0 3.903-2.815 5.842 5.842 0 0 0-.728-6.877v.002Zm-8.806 12.233a4.39 4.39 0 0 1-2.81-1.01 2.79 2.79 0 0 0 .138-.078l4.665-2.678a.754.754 0 0 0 .384-.66v-6.537l1.971 1.132a.07.07 0 0 1 .038.054v5.413c-.003 2.407-1.964 4.359-4.386 4.364Zm-9.432-4.005a4.329 4.329 0 0 1-.523-2.923l.138.082 4.665 2.678a.765.765 0 0 0 .767 0l5.694-3.27v2.264a.07.07 0 0 1-.028.06l-4.715 2.707a4.41 4.41 0 0 1-5.997-1.598h-.001Zm-1.227-10.121a4.365 4.365 0 0 1 2.285-1.913l-.003.16v5.356a.754.754 0 0 0 .383.66l5.695 3.268-1.972 1.131a.069.069 0 0 1-.066.006l-4.716-2.708a4.355 4.355 0 0 1-1.607-5.96h.001Zm16.197 3.747-5.694-3.269 1.971-1.13a.071.071 0 0 1 .067-.007l4.716 2.707a4.351 4.351 0 0 1 1.606 5.962 4.38 4.38 0 0 1-2.284 1.913v-5.517a.753.753 0 0 0-.381-.66h-.001Zm1.962-2.936a5.658 5.658 0 0 0-.138-.082l-4.665-2.678a.763.763 0 0 0-.766 0l-5.695 3.269v-2.263a.074.074 0 0 1 .028-.06l4.715-2.705c2.1-1.204 4.786-.487 5.996 1.601.512.882.697 1.915.524 2.918h.001Zm-12.336 4.034-1.972-1.132a.07.07 0 0 1-.038-.054v-5.413c.002-2.41 1.969-4.363 4.393-4.362 1.026 0 2.019.358 2.807 1.01a3.12 3.12 0 0 0-.138.078l-4.665 2.678a.752.752 0 0 0-.384.66l-.003 6.533v.002Zm1.071-2.295 2.537-1.456 2.537 1.455v2.912l-2.537 1.455-2.537-1.456v-2.91Z"
      />
      <mask
        id="a"
        width={108}
        height={13}
        x={0}
        y={14}
        maskUnits="userSpaceOnUse"
        style={{
          maskType: "luminance",
        }}
      >
        <path fill="#fff" d="M0 14.272h107.416v12.019H0V14.272Z" />
      </mask>
      <g mask="url(#a)">
        <path
          fill="#fff"
          d="m21.796 22.61-5.228-8.135h-2.822v11.62h2.407v-8.134l5.228 8.135h2.822v-11.62h-2.407v8.133Z"
        />
      </g>
      <mask
        id="b"
        width={108}
        height={13}
        x={0}
        y={14}
        maskUnits="userSpaceOnUse"
        style={{
          maskType: "luminance",
        }}
      >
        <path fill="#fff" d="M0 14.272h107.416v12.019H0V14.272Z" />
      </mask>
      <g mask="url(#b)">
        <path
          fill="#fff"
          d="M26.193 16.716h3.9v9.38h2.49v-9.38h3.901v-2.24h-10.29v2.24Z"
        />
      </g>
      <mask
        id="c"
        width={108}
        height={13}
        x={0}
        y={14}
        maskUnits="userSpaceOnUse"
        style={{
          maskType: "luminance",
        }}
      >
        <path fill="#fff" d="M0 14.272h107.416v12.019H0V14.272Z" />
      </mask>
      <g mask="url(#c)">
        <path
          fill="#fff"
          d="M46.449 19.107h-5.477v-4.632h-2.49v11.62h2.49v-4.747h5.477v4.748h2.49v-11.62h-2.49v4.63Z"
        />
      </g>
      <mask
        id="d"
        width={108}
        height={13}
        x={0}
        y={14}
        maskUnits="userSpaceOnUse"
        style={{
          maskType: "luminance",
        }}
      >
        <path fill="#fff" d="M0 14.272h107.416v12.019H0V14.272Z" />
      </mask>
      <g mask="url(#d)">
        <path
          fill="#fff"
          d="M54.5 16.716h3.072c1.229 0 1.876.448 1.876 1.295s-.647 1.295-1.876 1.295h-3.071v-2.59Zm7.438 1.295c0-2.191-1.61-3.536-4.25-3.536h-5.677v11.62h2.49v-4.548h2.772l2.49 4.549h2.756L59.762 21.2c1.384-.532 2.176-1.653 2.176-3.19Z"
        />
      </g>
      <mask
        id="e"
        width={108}
        height={13}
        x={0}
        y={14}
        maskUnits="userSpaceOnUse"
        style={{
          maskType: "luminance",
        }}
      >
        <path fill="#fff" d="M0 14.272h107.416v12.019H0V14.272Z" />
      </mask>
      <g mask="url(#e)">
        <path
          fill="#fff"
          d="M69.567 23.967c-1.958 0-3.153-1.395-3.153-3.669 0-2.307 1.195-3.702 3.153-3.702 1.942 0 3.12 1.395 3.12 3.702 0 2.274-1.178 3.669-3.12 3.669Zm0-9.695c-3.352 0-5.726 2.49-5.726 6.026 0 3.503 2.374 5.993 5.726 5.993 3.336 0 5.692-2.49 5.692-5.993 0-3.536-2.356-6.026-5.692-6.026Z"
        />
      </g>
      <mask
        id="f"
        width={108}
        height={13}
        x={0}
        y={14}
        maskUnits="userSpaceOnUse"
        style={{
          maskType: "luminance",
        }}
      >
        <path fill="#fff" d="M0 14.272h107.416v12.019H0V14.272Z" />
      </mask>
      <g mask="url(#f)">
        <path
          fill="#fff"
          d="M83.15 19.638H80.08v-2.922h3.072c1.228 0 1.876.498 1.876 1.461 0 .963-.648 1.46-1.876 1.46Zm.117-5.163h-5.68v11.62h2.492V21.88h3.188c2.64 0 4.251-1.394 4.251-3.702 0-2.307-1.61-3.702-4.251-3.702Z"
        />
      </g>
      <mask
        id="g"
        width={108}
        height={13}
        x={0}
        y={14}
        maskUnits="userSpaceOnUse"
        style={{
          maskType: "luminance",
        }}
      >
        <path fill="#fff" d="M0 14.272h107.416v12.019H0V14.272Z" />
      </mask>
      <g mask="url(#g)">
        <path
          fill="#fff"
          d="M104.378 22.19c-.432 1.13-1.295 1.777-2.473 1.777-1.959 0-3.154-1.395-3.154-3.669 0-2.307 1.195-3.702 3.154-3.702 1.178 0 2.041.648 2.473 1.776h2.639c-.648-2.49-2.589-4.1-5.112-4.1-3.353 0-5.726 2.49-5.726 6.026 0 3.503 2.373 5.993 5.726 5.993 2.539 0 4.481-1.627 5.128-4.1h-2.655Z"
        />
      </g>
      <mask
        id="h"
        width={108}
        height={13}
        x={0}
        y={14}
        maskUnits="userSpaceOnUse"
        style={{
          maskType: "luminance",
        }}
      >
        <path fill="#fff" d="M0 14.272h107.416v12.019H0V14.272Z" />
      </mask>
      <g mask="url(#h)">
        <path
          fill="#fff"
          d="m88.51 14.475 4.633 11.62h2.54l-4.633-11.62h-2.54Z"
        />
      </g>
      <mask
        id="i"
        width={108}
        height={13}
        x={0}
        y={14}
        maskUnits="userSpaceOnUse"
        style={{
          maskType: "luminance",
        }}
      >
        <path fill="#fff" d="M0 14.272h107.416v12.019H0V14.272Z" />
      </mask>
      <g mask="url(#i)">
        <path
          fill="#fff"
          d="m4.375 21.497 1.585-4.084 1.586 4.084h-3.17Zm.257-7.022L0 26.095h2.59l.947-2.44h4.847l.947 2.44h2.59l-4.632-11.62H4.632Z"
        />
      </g>
      <path
        fill="#fff"
        d="M338.896 24.83a2.712 2.712 0 0 0-.37-.818c.036-.132.063-.266.082-.401.119-.86-.179-1.646-.741-2.255a2.588 2.588 0 0 0-.975-.686 13.028 13.028 0 0 0 .192-4.952 13.145 13.145 0 0 0-.878-3.103 13.23 13.23 0 0 0-5.275-6.1 13.032 13.032 0 0 0-6.788-1.893c-7.231 0-13.091 5.86-13.091 13.09a12.954 12.954 0 0 0 .348 2.995 2.579 2.579 0 0 0-.885.65c-.561.607-.859 1.391-.74 2.25.018.137.046.272.082.405-.168.25-.293.527-.37.819-.173.655-.116 1.247.103 1.765a2.595 2.595 0 0 0 .132 2.025c.226.458.549.812.947 1.129.473.376 1.066.696 1.781 1.003.853.364 1.894.706 2.367.83 1.223.317 2.396.518 3.584.528 1.694.016 3.152-.382 4.196-1.401a12.66 12.66 0 0 0 1.546.093 13.469 13.469 0 0 0 1.633-.102c1.041 1.025 2.505 1.427 4.205 1.41 1.188-.009 2.361-.21 3.58-.526a20.573 20.573 0 0 0 2.371-.831c.715-.308 1.307-.627 1.784-1.003.395-.317.718-.671.944-1.129.314-.627.37-1.348.135-2.026.217-.518.273-1.11.101-1.765Zm-1.213 1.722c.24.456.255.971.043 1.45-.321.728-1.12 1.3-2.671 1.916-.964.382-1.847.627-1.855.629-1.276.33-2.429.499-3.428.499-1.654 0-2.884-.457-3.665-1.358a12.283 12.283 0 0 1-3.989.023c-.781.886-2.005 1.335-3.645 1.335-.999 0-2.152-.168-3.428-.5a20.54 20.54 0 0 1-1.855-.628c-1.551-.615-2.35-1.188-2.671-1.915a1.634 1.634 0 0 1 .115-1.575 1.835 1.835 0 0 1-.248-1.487 1.66 1.66 0 0 1 .56-.88 1.83 1.83 0 0 1-.249-.694c-.077-.534.1-1.067.499-1.502.311-.338.75-.524 1.236-.524h.012a12.13 12.13 0 0 1-.55-3.629c0-6.71 5.439-12.15 12.151-12.15 6.711 0 12.151 5.44 12.151 12.15a12.128 12.128 0 0 1-.555 3.638c.059-.006.116-.009.173-.009.486 0 .925.186 1.235.525.399.434.576.968.499 1.502a1.83 1.83 0 0 1-.249.693c.268.216.465.518.561.88a1.845 1.845 0 0 1-.249 1.487c.026.04.05.081.072.124Z"
      />
      <path
        fill="#FF9D00"
        d="M337.611 26.429a1.845 1.845 0 0 0 .249-1.487 1.668 1.668 0 0 0-.561-.881 1.83 1.83 0 0 0 .249-.694c.077-.534-.1-1.067-.499-1.502a1.65 1.65 0 0 0-1.235-.524c-.057 0-.114.003-.173.009.368-1.178.555-2.404.554-3.637 0-6.71-5.44-12.151-12.151-12.151-6.71 0-12.151 5.44-12.151 12.15-.001 1.23.185 2.454.551 3.629h-.013a1.65 1.65 0 0 0-1.235.524c-.399.435-.576.968-.499 1.502.035.246.12.482.249.694a1.667 1.667 0 0 0-.561.88 1.836 1.836 0 0 0 .249 1.488 1.637 1.637 0 0 0-.115 1.575c.321.726 1.12 1.299 2.671 1.914.964.382 1.847.627 1.855.63 1.276.33 2.429.498 3.428.498 1.64 0 2.864-.449 3.645-1.335 1.322.21 2.67.202 3.989-.023.781.902 2.011 1.358 3.665 1.358.999 0 2.152-.168 3.428-.499.008-.002.891-.247 1.855-.63 1.551-.614 2.35-1.187 2.671-1.914a1.637 1.637 0 0 0-.115-1.574Zm-16.052 1.895c-.067.117-.14.232-.216.344a2.196 2.196 0 0 1-.776.69c-.59.321-1.336.433-2.094.433-1.197 0-2.425-.28-3.113-.458-.034-.01-4.218-1.19-3.688-2.197.089-.169.236-.236.421-.236.745 0 2.103 1.11 2.687 1.11.13 0 .222-.055.26-.19.249-.893-3.78-1.268-3.441-2.56.06-.228.222-.321.451-.321.986 0 3.199 1.734 3.661 1.734.036 0 .061-.01.075-.032l.006-.01c.217-.359.093-.62-1.396-1.531l-.143-.087c-1.637-.992-2.787-1.588-2.133-2.3.075-.082.182-.118.311-.118.154 0 .34.051.545.137.865.365 2.063 1.36 2.564 1.794.079.068.157.137.235.207 0 0 .634.66 1.017.66a.233.233 0 0 0 .214-.122c.272-.458-2.525-2.578-2.683-3.452-.107-.593.075-.893.412-.893.16 0 .355.068.571.205.668.424 1.959 2.643 2.432 3.506.158.289.429.41.672.41.484 0 .862-.48.044-1.09-1.228-.92-.797-2.422-.211-2.514a.487.487 0 0 1 .076-.006c.533 0 .768.918.768.918s.689 1.731 1.873 2.914c1.075 1.074 1.225 1.955.599 3.055Zm3.828.202-.061.008-.105.012-.165.016-.054.005-.049.004-.07.006-.078.005-.077.005-.017.001-.06.004-.026.001a5.78 5.78 0 0 1-.072.004l-.084.003-.075.003-.051.001h-.026l-.047.001h-.025l-.047.001h-.155l-.211-.001-.057-.002h-.048l-.061-.003-.074-.002-.068-.003-.017-.001-.065-.004c-.018 0-.035-.001-.053-.003l-.042-.002a11.674 11.674 0 0 1-.156-.011l-.055-.005-.069-.005a5.49 5.49 0 0 1-.215-.021h-.003c.657-1.466.325-2.835-1.003-4.162-.87-.87-1.45-2.154-1.57-2.436-.243-.835-.887-1.762-1.956-1.762-.091 0-.181.007-.27.021-.469.074-.878.343-1.17.75-.316-.394-.623-.706-.9-.882-.419-.265-.836-.4-1.243-.4-.508 0-.962.209-1.278.587l-.008.01-.018-.075v-.004a10.276 10.276 0 0 1-.152-.778l-.001-.005-.009-.06-.025-.175-.01-.08-.01-.08-.009-.075v-.007a10.787 10.787 0 0 1-.031-.31l-.003-.04-.005-.068-.004-.056-.001-.013-.009-.156-.004-.082-.003-.071-.001-.022-.002-.065-.001-.056-.002-.067-.001-.07-.001-.07v-.071c0-6.018 4.879-10.896 10.897-10.896 6.018 0 10.896 4.878 10.896 10.896v.14l-.001.07-.001.059a.863.863 0 0 1-.002.051l-.002.066v.002l-.003.076c-.001.023-.001.045-.003.067v.016l-.004.071c-.009.153-.02.306-.035.458v.002l-.007.075-.007.06-.013.115-.007.058-.009.07-.01.074a2.389 2.389 0 0 1-.013.084l-.009.066-.012.075-.013.074-.013.074-.027.149-.045.22-.016.074-.016.073a1.64 1.64 0 0 0-1.161-.46c-.406 0-.824.134-1.242.399-.278.176-.584.489-.9.881-.293-.406-.702-.675-1.17-.749a1.742 1.742 0 0 0-.27-.021c-1.07 0-1.713.928-1.957 1.762-.121.282-.7 1.566-1.572 2.437-1.327 1.323-1.661 2.686-1.015 4.145Zm11.25-2.934-.005.013a.664.664 0 0 1-.072.143c-.024.036-.051.07-.079.102l-.021.022-.031.033c-.194.192-.489.36-.823.514l-.114.052-.039.017-.078.034a3.785 3.785 0 0 1-.08.033l-.08.033c-.188.077-.38.15-.567.223l-.08.032-.08.03-.156.062-.076.031-.075.03-.037.016-.073.03c-.551.238-.948.478-.865.777l.008.025a.29.29 0 0 0 .056.095c.098.102.277.086.502.005l.094-.036.02-.008a4.09 4.09 0 0 0 .161-.073l.042-.02c.275-.135.587-.316.897-.481a8.28 8.28 0 0 1 .376-.19c.293-.138.569-.236.79-.236.104 0 .196.021.272.07l.013.009a.417.417 0 0 1 .136.157c.109.208.017.423-.197.634-.206.203-.526.401-.889.585l-.082.04c-1.083.532-2.5.933-2.52.938-.378.098-.918.226-1.527.323l-.09.014-.015.002a8.793 8.793 0 0 1-.415.054l-.013.002c-.256.03-.513.049-.77.058h-.004a7.867 7.867 0 0 1-.279.005h-.108a6.605 6.605 0 0 1-.425-.019h-.01a4.35 4.35 0 0 1-.412-.05 6.624 6.624 0 0 1-.104-.017l-.048-.009h-.003c-.05-.01-.1-.02-.149-.032l-.086-.02-.017-.005-.042-.01-.007-.003-.045-.013-.048-.014-.006-.001-.041-.013-.048-.015-.038-.013-.029-.01a3.784 3.784 0 0 1-.081-.03l-.026-.01-.021-.009a3.093 3.093 0 0 1-.121-.051l-.027-.013-.004-.002-.029-.013-.055-.027-.005-.003-.027-.014a2.23 2.23 0 0 1-.138-.078l-.025-.016-.037-.023-.032-.021-.034-.024-.021-.015a1.62 1.62 0 0 1-.064-.048l-.034-.026-.04-.033-.032-.028h-.001a.787.787 0 0 1-.034-.031.79.79 0 0 1-.034-.032h-.001a.768.768 0 0 1-.067-.068 1.874 1.874 0 0 1-.033-.035l-.03-.034-.003-.004a1.341 1.341 0 0 1-.146-.189l-.018-.027a11.882 11.882 0 0 1-.117-.181l-.03-.049-.004-.007-.028-.047a.267.267 0 0 1-.014-.025l-.015-.027-.009-.015-.005-.009-.028-.052a.23.23 0 0 0-.013-.023l-.013-.025-.013-.024a3.605 3.605 0 0 1-.091-.197l-.011-.024a.895.895 0 0 0-.019-.049l-.009-.023c-.015-.04-.03-.08-.043-.12l-.014-.044a3.2 3.2 0 0 1-.055-.215l-.004-.024a1.143 1.143 0 0 1-.018-.116c-.002-.007-.002-.015-.003-.023l-.003-.023-.007-.092-.001-.023-.001-.045c-.008-.615.303-1.205.968-1.87 1.184-1.183 1.873-2.913 1.873-2.913s.018-.073.057-.178l.017-.045a2.13 2.13 0 0 1 .076-.172l.006-.01a1.54 1.54 0 0 1 .078-.14l.02-.032a1.63 1.63 0 0 1 .108-.138.614.614 0 0 1 .252-.177l.011-.004.023-.007a.525.525 0 0 1 .027-.006h.004a.418.418 0 0 1 .057-.008h.001l.03-.001h.038a.516.516 0 0 1 .334.183.857.857 0 0 1 .138.209l.018.037a1.457 1.457 0 0 1 .125.562v.12c-.008.186-.046.37-.113.545l-.02.048a1.121 1.121 0 0 1-.043.095l-.037.072-.027.047c-.023.04-.048.078-.075.117l-.016.024a2.237 2.237 0 0 1-.456.46 1.958 1.958 0 0 0-.243.212c-.216.227-.266.427-.218.579a.355.355 0 0 0 .082.137l.008.008.007.008a.417.417 0 0 0 .025.022l.009.007c.021.016.043.03.066.043l.02.01a.605.605 0 0 0 .077.032l.023.007.009.002.013.004.011.002.012.003.012.003.011.001.026.004.008.002.015.001.009.001.015.001H329.878l.016-.001.02-.002.019-.002.013-.002a.72.72 0 0 0 .168-.046l.023-.01a.694.694 0 0 0 .204-.133l.02-.018a.825.825 0 0 0 .147-.196c.282-.512.577-1.016.885-1.512l.042-.067.043-.068.064-.102.022-.034c.071-.112.144-.223.218-.334l.044-.065c.088-.13.176-.257.262-.378l.044-.06a7.83 7.83 0 0 1 .295-.382l.041-.049a.338.338 0 0 1 .02-.024l.04-.046.02-.022.038-.044.019-.02.057-.06.038-.039c.073-.074.153-.141.24-.199l.02-.013a.689.689 0 0 1 .059-.036c.34-.193.622-.207.784-.045.098.098.152.26.149.487l-.001.03v.011l-.001.032c0 .013-.002.025-.003.038l-.003.034-.001.01-.004.03-.001.01-.006.04-.006.038-.004.022a.622.622 0 0 1-.055.17 1.53 1.53 0 0 1-.12.217l-.052.078a3.943 3.943 0 0 1-.141.19l-.023.029c-.08.1-.162.197-.247.293l-.027.03a7.49 7.49 0 0 1-.111.122l-.028.03-.058.062-.058.063-.059.062-.06.063-.061.062-.122.126c-.583.598-1.197 1.194-1.405 1.566a.985.985 0 0 0-.038.074.309.309 0 0 0-.033.172c.003.016.008.03.016.044a.303.303 0 0 0 .066.076c.043.03.095.046.148.045h.016l.017-.001.017-.002.014-.002.005-.001.013-.003h.003l.015-.004h.005l.015-.005.018-.005a.719.719 0 0 0 .112-.044l.019-.01.019-.009c.046-.024.091-.05.134-.077l.02-.013.019-.013.019-.013.01-.007.028-.02c.025-.017.049-.035.074-.054l.002-.002.038-.03c.053-.041.103-.083.149-.123l.03-.026.003-.003.015-.014c.038-.033.071-.064.098-.09l.011-.01.028-.026.016-.016.005-.006.002-.001.017-.017.011-.011.001-.001.005-.005.006-.006.002-.002.005-.005.03-.025.016-.015.026-.023.02-.018.01-.009.021-.018.03-.026.016-.014.226-.194.036-.03.059-.05.06-.05c.08-.067.165-.136.255-.209l.059-.047.154-.122.064-.05a21.741 21.741 0 0 1 .541-.4l.056-.039.115-.08.035-.023a8.41 8.41 0 0 1 .209-.137l.035-.022.035-.021.104-.064.034-.02.069-.04.067-.039.014-.007.053-.03.067-.035.033-.017.032-.016.033-.016a3.216 3.216 0 0 1 .281-.121l.059-.021.051-.017.006-.001a.457.457 0 0 1 .027-.008h.002l.055-.015h.001a1.184 1.184 0 0 1 .227-.03h.012c.016 0 .032.001.047.003l.021.002h.003l.021.003.02.004h.002l.02.006a.369.369 0 0 1 .147.08l.004.005a.042.042 0 0 1 .007.007l.007.007c.058.06.106.13.142.206l.005.013a.471.471 0 0 1 .006.367.65.65 0 0 1-.048.103 1.197 1.197 0 0 1-.158.208l-.013.014-.06.06-.028.028-.03.027-.015.013a2.563 2.563 0 0 1-.114.097l-.071.055a6.315 6.315 0 0 1-.417.294c-.132.086-.266.171-.401.253-.283.174-.596.36-.93.563l-.087.052a24.33 24.33 0 0 0-.268.166l-.043.026-.08.052-.159.103-.043.028a2.298 2.298 0 0 0-.062.04l-.021.014-.062.042-.033.022-.038.027-.036.025c-.06.042-.116.083-.167.121l-.019.015-.09.07c-.044.035-.085.07-.121.102l-.018.017-.03.028-.02.019-.009.009-.06.063-.009.011a.73.73 0 0 0-.056.071l-.007.01a.477.477 0 0 0-.035.057l-.008.015-.007.015-.005.01-.003.007-.002.007-.003.009a.39.39 0 0 0-.019.075l-.001.009-.001.008v.045l.001.012.001.007.001.011.003.016.003.016.004.016.01.03.007.018.001.003.006.012.007.016a.785.785 0 0 0 .025.048l.01.017.01.016a.03.03 0 0 0 .005.008l.004.003.003.004.004.002a.056.056 0 0 0 .018.01.077.077 0 0 0 .01.002c.083.019.253-.05.48-.17l.04-.02.069-.039.034-.018.073-.042.046-.026c.3-.172.657-.393 1.027-.61l.104-.06.07-.045a17.13 17.13 0 0 1 .556-.303l.069-.034a6.958 6.958 0 0 1 .396-.182l.048-.02.005-.002c.256-.101.487-.164.674-.164a.66.66 0 0 1 .121.01h.001a.687.687 0 0 1 .037.008h.002a.37.37 0 0 1 .23.163c.02.03.036.064.048.1l.012.04a.56.56 0 0 1-.008.332Z"
      />
      <path
        fill="#FFD21E"
        fillRule="evenodd"
        d="M334.941 17.783v-.07c0-6.018-4.877-10.896-10.895-10.896s-10.897 4.878-10.897 10.895V17.784l.001.07.001.052v.019l.001.026.001.04.001.057.002.065.001.022.003.068v.003l.004.08v.002l.005.081.004.075.001.005.004.064v.006l.005.062.001.004.003.036c.008.103.018.207.03.31v.007l.009.076.01.08.007.054.003.025.025.176.001.004.008.055c.041.263.092.524.153.784v.003l.005.02.013.055.008-.01c.316-.378.77-.587 1.278-.587.407 0 .824.135 1.243.4.277.176.584.489.9.881.292-.406.701-.675 1.17-.749.089-.014.179-.021.27-.021 1.069 0 1.713.928 1.956 1.762.12.282.7 1.566 1.573 2.435 1.328 1.327 1.66 2.696 1.003 4.161h.002c.045.006.09.01.135.014l.081.008h.01l.059.006.055.004.156.011.042.002.033.002.02.002.064.003h.018l.067.004.075.002.061.002h.01a.77.77 0 0 0 .038.002h.014c.085.002.169.003.254.003h.154c.016-.002.032-.002.048-.002h.098l.05-.002.076-.003.084-.003.072-.004.025-.001.039-.002.022-.002h.017l.077-.006.077-.005.07-.006.05-.004.054-.005c.09-.008.18-.018.27-.028l.061-.008c-.647-1.46-.312-2.822 1.011-4.144.871-.87 1.451-2.155 1.571-2.437.244-.834.887-1.762 1.957-1.762.09 0 .181.007.27.021.468.074.877.343 1.17.75.316-.393.623-.706.9-.882.418-.265.836-.4 1.243-.4.448 0 .854.163 1.16.46l.017-.072.016-.073.016-.08a6.542 6.542 0 0 0 .055-.29l.005-.026.008-.048.013-.074.002-.01.01-.065.01-.066.021-.147.001-.012.009-.068.008-.06.012-.115.006-.046.001-.013.007-.075v-.002l.006-.06c.01-.113.019-.227.026-.341l.003-.057.004-.071.001-.016.006-.143V18.081a.745.745 0 0 0 .002-.048v-.006l.001-.045.001-.013v-.045l.001-.013.001-.058v-.07Zm-13.598 10.885c.863-1.265.802-2.215-.382-3.398-1.184-1.184-1.873-2.915-1.873-2.915s-.257-1.005-.844-.912c-.587.092-1.017 1.594.211 2.513 1.229.92-.244 1.543-.717.68-.472-.862-1.763-3.08-2.432-3.505-.669-.424-1.14-.186-.982.688.078.434.807 1.175 1.487 1.867.691.703 1.333 1.355 1.195 1.585-.272.459-1.23-.538-1.23-.538s-3.002-2.731-3.655-2.02c-.603.656.326 1.214 1.758 2.074l.375.226c1.638.991 1.765 1.253 1.533 1.628-.086.139-.634-.19-1.308-.595-1.15-.69-2.666-1.6-2.88-.786-.185.705.93 1.137 1.942 1.529.842.326 1.613.624 1.499 1.03-.117.42-.754.07-1.45-.314-.781-.43-1.638-.9-1.918-.37-.529 1.006 3.654 2.189 3.688 2.198 1.352.35 4.784 1.093 5.983-.665Zm5.559 0c-.862-1.265-.801-2.215.383-3.398 1.184-1.184 1.873-2.915 1.873-2.915s.257-1.005.844-.912c.586.092 1.017 1.594-.211 2.513-1.229.92.244 1.543.717.68.472-.862 1.762-3.08 2.431-3.505.67-.424 1.14-.186.983.688-.079.434-.807 1.175-1.488 1.867-.691.703-1.332 1.355-1.195 1.585.272.459 1.231-.538 1.231-.538s3.001-2.732 3.655-2.02c.602.656-.327 1.214-1.758 2.073-.125.076-.251.15-.375.226-1.638.992-1.766 1.253-1.533 1.628.086.14.634-.19 1.308-.595 1.15-.69 2.665-1.6 2.88-.785.185.704-.93 1.136-1.942 1.528-.842.327-1.613.625-1.5 1.03.118.42.754.07 1.45-.313.782-.43 1.638-.902 1.918-.37.53 1.006-3.654 2.188-3.688 2.197-1.351.351-4.783 1.094-5.983-.664Z"
        clipRule="evenodd"
      />
      <path
        fill="#32343D"
        fillRule="evenodd"
        d="M327.585 14.747c.169.06.296.243.416.417.163.235.314.454.546.33a1.568 1.568 0 1 0-2.121-.647c.108.202.347.106.599.006.197-.08.403-.162.56-.106Zm-7.387 0c-.17.06-.296.243-.417.417-.162.235-.313.454-.545.33a1.564 1.564 0 0 1-.716-1.981 1.565 1.565 0 0 1 1.904-.903 1.572 1.572 0 0 1 1.108 1.344c.031.308-.03.619-.176.893-.107.202-.346.106-.598.006-.198-.08-.403-.162-.56-.106Zm6.244 6.722c1.171-.923 1.601-2.429 1.601-3.356 0-.733-.494-.503-1.283-.112l-.045.022c-.725.36-1.69.837-2.749.837-1.06 0-2.025-.478-2.75-.837-.815-.404-1.326-.657-1.326.09 0 .957.457 2.528 1.713 3.441a2.734 2.734 0 0 1 1.664-1.407c.125-.037.255.179.387.4.127.213.258.43.39.43.141 0 .28-.214.416-.424.141-.22.28-.433.413-.39a2.728 2.728 0 0 1 1.569 1.306Z"
        clipRule="evenodd"
      />
      <path
        fill="#FF323D"
        d="M326.441 21.468c-.609.48-1.42.803-2.475.803-.992 0-1.767-.284-2.363-.717a2.733 2.733 0 0 1 1.664-1.407c.246-.074.507.83.777.83.289 0 .568-.898.829-.814a2.727 2.727 0 0 1 1.568 1.305Z"
      />
      <path
        fill="#FFAD03"
        fillRule="evenodd"
        d="M317.398 16.051a1.013 1.013 0 0 1-.956.094 1.016 1.016 0 0 1-.457-1.507 1.02 1.02 0 1 1 1.413 1.413Zm14.581 0a1.013 1.013 0 0 1-.956.094 1.02 1.02 0 1 1 .956-.094Z"
        clipRule="evenodd"
      />
    </g>
  </svg>
);
