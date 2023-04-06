import Header from "src/shared/Header";
import Container from "src/shared/layout/Container";
import ComparisonTable from "src/shared/Pricing/ComparisionTable";
import { FAQRow } from "src/shared/Pricing/FAQ";
import PlanCard from "src/shared/Pricing/PlanCard";
import Footer from "src/shared/Footer";

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

export default function Ocoya() {
  return (
    <div className="font-sans">
      <Header />
      <div
        style={{
        }}
      >
        <Container className="text-white">
        <div className="flex items-center">
          <div className="max-w-[760px] pb-20 pr-40">
            <p className="text-white mt-24 text-sm ml-1">Case study + partership</p>

            <img src="/img/ocoya.svg" alt="Ocoya" width="320" className="mt-8 mb-12" />

            <p>Within two years, over 50,000 users — including the worlds biggest companies like Pepsi and WPP — use <a href="https://www.ocoya.com" rel="nofollow" target="_blank" className="text-white">Ocoya</a> to manage their social media&nbsp;marketing.</p>

            <p className="mt-4">Learn how Ocoya uses Inngest to develop and deliver their world class product in record time, with end-to-end local testing.</p>
          </div>

          <div className="flex-initial">
            <img src="https://cdn.arcade.software/cdn-cgi/image/fit=scale-down,format=auto,width=3840/extension-uploads/f3e1955b-ff3e-4d1b-b889-f1e18c963f8a.png" alt="Ocoya UI" width="800"
            className="mt-24"
            />
          </div>
</div>

        <div className="m-auto max-w-3xl pt-24">
          <div className="">
            <h2 className="text-xl lg:text-3xl text-white mb-8 font-semibold tracking-tight">
              Ocoya:  Workflows + Queues
            </h2>

            <p>Every aspect of Ocoya requires complex workflows, from scheduling social media content to ecommerce imports.  Traditionally, developing this functionality requires setting up multiple queues, dead-letter queues, services, subscribers, and backoffs, along with code for delivering to each queue.</p>
            <p className="mt-4">Only a subset of their engineering team could handle queues & infra, and it wasn’t locally testable.  Plus, code was also split over many codebases, making debugging or changes difficult.</p>

            <p className="mt-12 font-semibold">Fixing problems: out with the old, in with the new.</p>

            <p className="mt-4">When planning and designing their ecommerce product range, Ocoya wanted to <strong>simplify and speed up development across their entire team</strong>.  Using Inngest, Ocoya was able to write their business logic directly as serverless functions without worrying about queues.  This allowed them to:</p>

            <ul className="list-disc my-4 ml-8 leading-7">
              <li>Speed up development of all business logic</li>
              <li>Enable local development for everyone in the team</li>
              <li>Simplify code into a single codebase, deploying reliable functions to Vercel</li>
              <li>Remove all queueing infrastructure</li>
              <li>Rely on the same CI/CD process via Vercel</li>
            </ul>

            <p>Additionally, using Inngest allows for easier debugging:  any failed functions are easily retryable, and the triggering event can be copied and ran locally to instantly replay functions in development.</p>

            <p className="mt-4 font-semibold">With just a few weeks of development, an entire new product category was planned, developed, and launched to production reliably, using Inngest, providing a better customer experience than ever before.</p>

          </div>
          <div className="mt-20">
            <h2 className="text-xl lg:text-3xl text-white mb-8 font-semibold tracking-tight">
              Moving forward
            </h2>

            <p>After implementing ecommerce imports and functionality in record time, both new and existing features can be refactored into this new way of working, unlocking better reliability, easier developing, faster debugging, and better performance.</p>

            <p className="mt-4">With the integration of Inngest, Ocoya can focus on their core product — delivering a world class product that enables users to deliver AI-enhanced social media and ecommerce content better than ever before.</p>
          </div>
          </div>
        </Container>
      </div>
      <Footer />
    </div>
  )
}
