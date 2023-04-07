import Header from "src/shared/Header";
import Container from "src/shared/layout/Container";
import Footer from "src/shared/Footer";
import CTACallout from "src/shared/CTACallout";

export async function getStaticProps() {
  return {
    props: {
      designVersion: "2",
      meta: {
        title: "Ocoya: A Case Study",
        description: "Simple pricing. Powerful functionality.",
      },
    },
  };
}

const proseBaseClasses = `prose prose-img:rounded-lg prose-code:bg-slate-800 prose-code:tracking-tight text-slate-200 prose-a:text-indigo-400 prose-a:no-underline hover:prose-a:underline hover:prose-a:text-white prose-a:font-medium prose-a:transition-all prose-invert`;

export default function Ocoya() {
  return (
    <div className="pt-8">
      <Header />
      <Container className="py-8">
        <div className="flex flex-col lg:flex-row gap-2 lg:gap-4 items-start lg:items-center">
          <h2 className="font-bold text-base text-white lg:border-r border-slate-600/50 pr-4">
            Case Study + Partnership
          </h2>
          <p className="text-slate-200 text-sm">Ocoya</p>
        </div>

        <div className="max-w-7xl mx-auto flex flex-col md:flex-row items-center gap-20 my-24">
          <div className="md:w-1/2">
            <img src="/img/ocoya.svg" alt="Ocoya" className="mb-12 w-80" />

            <div className={proseBaseClasses}>
              <p>
                Within two years, over 50,000 users — including the worlds
                biggest companies like Pepsi and WPP — use{" "}
                <a href="https://www.ocoya.com" rel="nofollow" target="_blank">
                  Ocoya
                </a>{" "}
                to manage their social media&nbsp;marketing.
              </p>

              <p>
                Learn how Ocoya uses Inngest to develop and deliver their world
                class product in record time, with end-to-end local testing.
              </p>
            </div>
          </div>

          <img
            src="https://cdn.arcade.software/cdn-cgi/image/fit=scale-down,format=auto,width=3840/extension-uploads/f3e1955b-ff3e-4d1b-b889-f1e18c963f8a.png"
            alt="Ocoya UI"
            className="w-[80%] md:w-1/2 md:max-w-xl rounded-md"
          />
        </div>

        <div className={`max-w-[74ch] m-auto mt-12 mb-20 ${proseBaseClasses}`}>
          <h2 className="text-xl lg:text-3xl text-white mb-8 font-semibold tracking-tight">
            Ocoya: Workflows + Queues
          </h2>

          <p>
            Every aspect of Ocoya requires complex workflows, from scheduling
            social media content to e-commerce imports. Traditionally,
            developing this functionality requires setting up multiple queues,
            dead-letter queues, services, subscribers, and backoffs, along with
            code for delivering to each queue.
          </p>
          <p>
            Only a subset of their engineering team could handle queues & infra,
            and it wasn't locally testable. Plus, code was also split over many
            codebases, making debugging or changes difficult.
          </p>

          <h3 className="text-lg font-semibold">
            Fixing problems: out with the old, in with the new.
          </h3>

          <p>
            When planning and designing their e-commerce product range, Ocoya
            wanted to{" "}
            <strong>
              simplify and speed up development across their entire team
            </strong>
            . Using Inngest, Ocoya was able to write their business logic
            directly as serverless functions without worrying about queues. This
            allowed them to:
          </p>

          <ul className="ml-8">
            <li>Speed up development of all business logic</li>
            <li>Enable local development for everyone in the team</li>
            <li>
              Simplify code into a single codebase, deploying reliable functions
              to Vercel
            </li>
            <li>Remove all queueing infrastructure</li>
            <li>Rely on the same CI/CD process via Vercel</li>
          </ul>

          <p>
            Additionally, using Inngest allows for easier debugging: any failed
            functions are easily retryable, and the triggering event can be
            copied and ran locally to instantly replay functions in development.
          </p>

          <p className="font-semibold">
            With just a few weeks of development, an entire new product category
            was planned, developed, and launched to production reliably, using
            Inngest, providing a better customer experience than ever before.
          </p>

          <figure className="my-12">
            <blockquote className="text-lg">
              At Ocoya, we were struggling with the complexities of managing our
              social media and e-commerce workflows. Thanks to Inngest, we were
              able to simplify our development process, speed up our time to
              market, and deliver a better customer experience. Inngest has
              become an essential tool in our tech stack, enabling us to focus
              on delivering a world-class product to our users.
            </blockquote>
            <figcaption className="mt-4 ml-6 flex flex-row items-center gap-4 text-slate-300 text-base">
              <img
                src="https://uploads-ssl.webflow.com/605dd4e52b25d35391c43725/62601246e6eb58a12097f7a2_profile_0.png"
                className="w-10"
                style={{ margin: 0 }}
              />
              <span>
                Aivaras Tumas, <cite>CEO & Co-founder &middot; Ocoya</cite>
              </span>
            </figcaption>
          </figure>

          <h2 className="text-xl lg:text-3xl text-white mb-8 font-semibold tracking-tight">
            Moving forward
          </h2>

          <p>
            After implementing e-commerce imports and functionality in record
            time, both new and existing features can be refactored into this new
            way of working, unlocking better reliability, easier developing,
            faster debugging, and better performance.
          </p>

          <p>
            With the integration of Inngest, Ocoya can focus on their core
            product — delivering a world class product that enables users to
            deliver AI-enhanced social media and e-commerce content better than
            ever before.
          </p>
        </div>

        <CTACallout
          text="Would your engineering team benefit from faster development time and a managed queuing solution?"
          cta={{
            href: "/contact?ref=case-study-ocoya",
            text: "Get in touch",
          }}
        />
      </Container>
      <Footer />
    </div>
  );
}
