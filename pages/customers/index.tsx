import Link from "next/link";
import Image from "next/image";
import clsx from "clsx";

import Header from "src/shared/Header";
import Container from "src/shared/layout/Container";
import Footer from "src/shared/Footer";
import CTACallout from "src/shared/CTACallout";
import { ArrowTopRightOnSquareIcon } from "@heroicons/react/20/solid";

import Button from "src/shared/legacy/Button";

export async function getStaticProps() {
  return {
    props: {
      designVersion: "2",
      meta: {
        title: "Customers",
        description: "Helping companies from startups to ",
      },
    },
  };
}

const featuredCompanies = [
  {
    src: "/assets/customers/tripadvisor.svg",
    name: "TripAdvisor",
    scale: 1.1,
  },
  {
    src: "/assets/customers/resend.svg",
    name: "Resend",
    scale: 0.8,
    href: "/customers/resend",
  },
  {
    src: "/assets/customers/ocoya.svg",
    name: "Ocoya",
    href: "/customers/ocoya",
  },
];

const grid = [
  {
    src: "/assets/customers/snaplet-dark.svg",
    name: "Snaplet",
    url: "https://www.snaplet.dev/",
    scale: 1.1,
    type: "company",
  },
  {
    type: "quote",
    name: "Resend",
    quote: {
      text: `The DX and visibility with Inngest is really incredible. We able to develop functions locally easier and faster that with our previous queue. Also, Inngest's tools give us the visibility to debug issues much quicker than before.`,
      attribution: {
        name: "Bu Kinoshita",
        title: "Co-founder",
      },
      avatar: "/assets/customers/resend-bu-kinoshita.jpg",
    },
  },
  {
    src: "/assets/customers/productlane.svg",
    name: "Productlane",
    url: "https://productlane.com/",
    scale: 1.4,
    type: "company",
  },
  {
    src: "/assets/customers/firstquadrant.svg",
    name: "FirstQuadrant.ai",
    url: "https://firstquadrant.ai/",
    scale: 1.2,
    type: "company",
  },
  {
    src: "/assets/customers/aomni-logo.svg",
    name: "Aomni",
    url: "https://www.aomni.com/",
    scale: 1,
    type: "company",
  },
  {
    src: "/assets/customers/zamp-logo.svg",
    name: "Zamp",
    url: "https://zamp.com/",
    scale: 1,
    type: "company",
  },
  {
    type: "quote",
    name: "Snaplet",
    quote: {
      text: `We switched from our PostgreSQL backed queue to Inngest in less than a day. Their approach is idiomatic with a great developer experience. Inngest allowed us to stop worrying about scalability and stability.`,
      attribution: {
        name: "Peter Pistorius",
        title: "CEO",
      },
      avatar: "/assets/customers/snaplet-peter-pistorius.png",
    },
  },
  {
    src: "/assets/customers/finta-logo.png?v=1",
    name: "Finta.io",
    url: "https://www.finta.io/",
    type: "company",
  },
  {
    src: "/assets/customers/secta-labs-logo.svg",
    name: "Secta.ai",
    url: "https://secta.ai/",
    type: "company",
  },
  {
    src: "/assets/customers/devjobs.svg",
    name: "DevJobs.at",
    url: "https://devjobs.at/",
    scale: 1.2,
    type: "company",
  },
  {
    src: "/assets/customers/niftykit.svg",
    name: "NiftyKit",
    url: "https://niftykit.com/",
    scale: 1,
    type: "company",
  },
  {
    type: "quote",
    name: "NiftyKit",
    quote: {
      text: `I can't stress enough how integral Inngest has been to our operations. It's more than just "battle tested" for us—it's been a game-changer and a cornerstone of our processes.`,
      attribution: {
        name: "Robin Curbelo",
        title: "Engineer",
      },
      avatar: "/assets/customers/niftykit-robin-curbelo.jpg",
    },
  },
  {
    src: "/assets/customers/lynq-logo.svg",
    name: "Lynq.ai",
    url: "https://www.lynq.ai/",
    scale: 1,
    type: "company",
  },
  {
    src: "/assets/customers/sliderule-analytics.png",
    name: "SlideRule",
    scale: 1,
    type: "company",
  },
  {
    src: "/assets/customers/double-logo.svg",
    name: "Double",
    scale: 1,
    type: "company",
  },
  {
    src: "/assets/customers/tono-logo.png",
    name: "Tono Health",
    scale: 0.9,
    type: "company",
  },
  {
    src: "/assets/customers/awaken-tax-logo.png",
    name: "Awaken.tax",
    type: "company",
  },
  {
    src: "/assets/customers/florian-works-logo.svg",
    name: "Florian Works",
    scale: 0.3,
    type: "company",
  },
];

export default function Customers() {
  return (
    <div className="pt-8 bg-slate-1000">
      <Header />
      <Container className="py-8">
        <div className="my-12 tracking-tight">
          <h1 className="max-w-5xl mx-auto text-center text-5xl font-semibold">
            <em>Our</em> customers deliver reliable products <br />
            for <em>their</em> customers
          </h1>
          <p className="max-w-lg mx-auto my-8 text-slate-400 text-center text-lg">
            From startups to public companies, these customers chose Inngest to
            power their products.
          </p>
        </div>
        <div className="max-w-6xl mx-auto">
          <div className="my-20 grid md:grid-cols-3 gap-6 md:gap-12">
            {featuredCompanies.map(({ src, name, href, scale = 1 }, idx) => (
              <div className="p-6 flex flex-col justify-items-start items-center border border-slate-100/10 rounded-2xl bg-slate-800/10 bg-[url(/assets/textures/wave.svg)]">
                <div className="mt-10 md:mt-20 h-20 md:h-24 flex items-center">
                  <Image
                    key={idx}
                    src={src}
                    alt={name}
                    title={name}
                    width={240 * 0.8 * scale}
                    height={120 * 0.8 * scale}
                    className={clsx(
                      "text-white mx-auto width-auto transition-all grayscale hover:grayscale-0",
                      `max-h-[${36 * scale}px]`
                    )}
                  />
                </div>
                <div className="h-10 md:h-20 flex items-center">
                  {!!href && (
                    <Link
                      href={href}
                      className="rounded-md px-3 py-1.5 text-sm bg-transparent transition-all text-white border border-slate-800 hover:border-slate-600 hover:bg-slate-500/10 whitespace-nowrap"
                    >
                      Read the case study →
                    </Link>
                  )}
                </div>
              </div>
            ))}
          </div>

          <div className="mt-12 mb-36 grid grid-cols-2 sm:grid-cols-2 lg:grid-cols-3 gap-6 md:gap-12">
            {grid.map(({ type, name, ...item }, idx) => {
              if (type === "quote") {
                const { quote } = item;
                return (
                  <blockquote className="mx-auto row-span-2 col-span-2 sm:col-span-1 my-8 max-w-3xl px-8 flex flex-col gap-8 bg-[url(/assets/textures/wave.svg)] bg-contain bg-center bg-no-repeat">
                    <p className="text-lg leading-7 relative">
                      <span className="absolute top-1 -left-4 text-2xl leading-3 text-slate-400/80">
                        &ldquo;
                      </span>
                      {quote.text}
                      <span className="ml-1 text-2xl leading-3 text-slate-400/80">
                        &rdquo;
                      </span>
                    </p>
                    <footer className="min-w-[180px] flex flex-row items-center gap-4">
                      <Image
                        src={quote.avatar}
                        alt={`Image of ${quote.attribution.name}`}
                        height="72"
                        width="72"
                        className="rounded-full h-12 w-12"
                      />
                      <cite className="text-slate-300 leading-8 not-italic">
                        <div className="text-base">
                          {quote.attribution.name}
                        </div>
                        <div className="text-sm text-slate-400">
                          {quote.attribution.title} - {name}
                        </div>
                      </cite>
                    </footer>
                  </blockquote>
                );
              }
              const { src, scale = 0.8 } = item;
              return (
                <div className="group relative px-6 py-8 h-full min-h-[148px] border border-slate-100/10 rounded-2xl flex items-center">
                  <Image
                    key={idx}
                    src={src}
                    alt={name}
                    title={name}
                    width={120 * scale}
                    height={30 * scale}
                    className={clsx(
                      "text-white m-auto width-auto transition-all grayscale hover:grayscale-0",
                      `max-h-[${36 * scale}px]`
                    )}
                  />
                  {!!item.url && (
                    <a
                      href={item.url}
                      target="_blank"
                      className="absolute hidden group-hover:flex flex-row gap-1 items-center bottom-2 right-2  px-3 py-1.5 text-xs bg-slate-400/10 hover:bg-slate-400/40 rounded-xl text-slate-50"
                    >
                      Visit website{" "}
                      <ArrowTopRightOnSquareIcon className="h-3" />
                    </a>
                  )}
                </div>
              );
            })}
          </div>
        </div>

        <CTACallout
          text="Would your engineering team benefit from faster development time and a managed workflow execution solution?"
          cta={{
            href: "/contact?ref=custoemrs",
            text: "Get in touch",
          }}
        />
      </Container>
      <Footer />
    </div>
  );
}
