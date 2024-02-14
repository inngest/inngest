import Image from 'next/image';
import Link from 'next/link';
import { ArrowTopRightOnSquareIcon } from '@heroicons/react/20/solid';
import clsx from 'clsx';
import CaseStudyCard from 'src/shared/CaseStudy/CaseStudyCard';
import Footer from 'src/shared/Footer';
import FooterCallout from 'src/shared/Footer/FooterCallout';
import Header from 'src/shared/Header';
import Container from 'src/shared/layout/Container';

export async function getStaticProps() {
  return {
    props: {
      designVersion: '2',
      meta: {
        title: 'Customer Case Studies',
        description:
          'From startups to public companies, these customers chose Inngest to build reliable products.',
      },
    },
  };
}

const caseStudies: {
  href: string;
  logo: string;
  name: string;
  title: string;
  snippet: React.ReactNode;
  // The use cases or industry for this customer
  tags?: string[];
}[] = [
  {
    href: '/customers/soundcloud',
    logo: '/assets/customers/soundcloud-logo-white.svg',
    name: 'SoundCloud',
    title: 'Streamlining dynamic video generation',
    snippet: 'Building scalable video pipelines with Inngest.',
    tags: ['Workflows'],
  },
  {
    href: '/customers/aomni',
    logo: '/assets/customers/aomni-logo.svg',
    name: 'Aomni',
    title: 'Productionizing AI-driven sales flows using serverless LLMs',
    snippet: 'Leveraging Inngest for production-grade complex state management and LLM chaining.',
    tags: ['AI'],
  },
  {
    href: '/customers/resend',
    logo: '/assets/customers/resend.svg',
    name: 'Resend',
    title: 'Scaling a developer email platform with serverless workflows',
    snippet: (
      <>
        "The <strong>DX and visibility</strong> with Inngest is really <em>incredible</em>."
      </>
    ),
    tags: ['Serverless'],
  },
  {
    href: '/customers/ocoya',
    logo: '/assets/customers/ocoya.svg',
    name: 'Ocoya',
    title: 'Shipping e-commerce import pipelines in record time',
    snippet: (
      <>
        "We were able to <strong>simplify</strong> our development process,{' '}
        <strong>speed up our time to market</strong>, and deliver a better customer experience."
      </>
    ),
    tags: ['Serverless'],
  },
  {
    href: '/customers/florian-works',
    logo: '/assets/customers/florian-works-logotype.svg',
    name: 'Florian Works',
    title: 'Building a mission-critical workflow engine on top of Inngest',
    snippet: (
      <>
        Saved <strong>months of development time</strong> and effort building on Inngest's
        primitives.
      </>
    ),
    tags: ['Workflow Engine'],
  },
];

const featuredCompanies = [
  {
    src: '/assets/customers/tripadvisor.svg',
    name: 'TripAdvisor',
    url: 'https://www.tripadvisor.com/',
    scale: 1.1,
  },
  {
    src: '/assets/customers/snaplet-dark.svg',
    name: 'Snaplet',
    url: 'https://www.snaplet.dev/',
    scale: 1.1,
    type: 'company',
  },
  {
    src: '/assets/customers/zamp-logo.svg',
    name: 'Zamp',
    url: 'https://zamp.com/',
    scale: 1,
    type: 'company',
  },
];

const grid = [
  {
    src: '/assets/customers/productlane.svg',
    name: 'Productlane',
    url: 'https://productlane.com/',
    scale: 1.4,
    type: 'company',
  },
  {
    type: 'quote',
    name: 'Snaplet',
    quote: {
      text: `We switched from our PostgreSQL backed queue to Inngest in less than a day. Their approach is idiomatic with a great developer experience. Inngest allowed us to stop worrying about scalability and stability.`,
      attribution: {
        name: 'Peter Pistorius',
        title: 'CEO',
      },
      avatar: '/assets/customers/snaplet-peter-pistorius.png',
    },
  },
  {
    src: '/assets/customers/leap-logo-white.svg',
    name: 'Leap',
    url: 'https://tryleap.ai/',
    scale: 1,
    type: 'company',
  },
  {
    src: '/assets/customers/firstquadrant.svg',
    name: 'FirstQuadrant.ai',
    url: 'https://firstquadrant.ai/',
    scale: 1.2,
    type: 'company',
  },
  {
    src: '/assets/customers/finta-logo.png?v=1',
    name: 'Finta.io',
    url: 'https://www.finta.io/',
    type: 'company',
  },
  {
    src: '/assets/customers/secta-labs-logo.svg',
    name: 'Secta.ai',
    url: 'https://secta.ai/',
    type: 'company',
  },
  {
    type: 'quote',
    name: 'NiftyKit',
    quote: {
      text: `I can't stress enough how integral Inngest has been to our operations. It's more than just "battle tested" for us—it's been a game-changer and a cornerstone of our processes.`,
      attribution: {
        name: 'Robin Curbelo',
        title: 'Engineer',
      },
      avatar: '/assets/customers/niftykit-robin-curbelo.jpg',
    },
  },
  {
    src: '/assets/customers/devjobs.svg',
    name: 'DevJobs.at',
    url: 'https://devjobs.at/',
    scale: 1.2,
    type: 'company',
  },
  {
    src: '/assets/customers/niftykit.svg',
    name: 'NiftyKit',
    url: 'https://niftykit.com/',
    scale: 1,
    type: 'company',
  },
  {
    src: '/assets/customers/lynq-logo.svg',
    name: 'Lynq.ai',
    url: 'https://www.lynq.ai/',
    scale: 1,
    type: 'company',
  },
  {
    src: '/assets/customers/sliderule-analytics.png',
    name: 'SlideRule',
    scale: 1,
    type: 'company',
  },
  {
    type: 'quote',
    name: 'Resend',
    quote: {
      text: `The DX and visibility with Inngest is really incredible. We able to develop functions locally easier and faster that with our previous queue. Also, Inngest's tools give us the visibility to debug issues much quicker than before.`,
      attribution: {
        name: 'Bu Kinoshita',
        title: 'Co-founder',
      },
      avatar: '/assets/customers/resend-bu-kinoshita.jpg',
    },
  },
  {
    src: '/assets/customers/double-logo.svg',
    name: 'Double',
    scale: 1,
    type: 'company',
  },
  {
    src: '/assets/customers/tono-logo.png',
    name: 'Tono Health',
    scale: 0.9,
    type: 'company',
  },
  {
    src: '/assets/customers/awaken-tax-logo.png',
    name: 'Awaken.tax',
    type: 'company',
  },
];

export default function Customers() {
  return (
    <div className="bg-slate-1000">
      <Header />
      <Container className="py-8">
        <div className="my-12 tracking-tight">
          <h1 className="mx-auto max-w-5xl text-center text-5xl font-semibold">
            <em>Our</em> customers deliver reliable products <br />
            for <em>their</em> customers
          </h1>
          <p className="mx-auto my-8 text-center text-lg text-slate-300">
            From startups to public companies, our customers chose Inngest to power their products.
          </p>
        </div>
        <div className="mx-auto max-w-6xl">
          <div className="my-20 grid gap-6 md:grid-cols-2 md:gap-12">
            {caseStudies.map((props, idx) => (
              <CaseStudyCard {...props} key={idx} />
            ))}
          </div>

          <div className="my-20 grid gap-6 md:grid-cols-3 md:gap-12">
            {featuredCompanies.map(({ src, name, scale = 1, url }, idx) => (
              <div className="group relative flex flex-col items-center justify-items-start rounded-2xl border border-slate-100/10 bg-slate-800/10 bg-[url(/assets/textures/wave.svg)] p-6">
                <div className="my-10 flex h-20 items-center md:my-20 md:h-24">
                  <Image
                    key={idx}
                    src={src}
                    alt={name}
                    title={name}
                    width={240 * 0.8 * scale}
                    height={120 * 0.8 * scale}
                    className={clsx(
                      'width-auto mx-auto text-white transition-all',
                      `max-h-[${36 * scale}px]`
                    )}
                  />
                </div>
                {!!url && (
                  <a
                    href={url}
                    target="_blank"
                    className="absolute bottom-2 right-2 hidden flex-row items-center gap-1 rounded-xl  bg-slate-400/10 px-3 py-1.5 text-xs text-slate-50 hover:bg-slate-400/40 group-hover:flex"
                  >
                    Visit website <ArrowTopRightOnSquareIcon className="h-3" />
                  </a>
                )}
              </div>
            ))}
          </div>

          <div className="mb-36 mt-12 grid grid-cols-2 gap-6 sm:grid-cols-2 md:gap-12 lg:grid-cols-3">
            {grid.map(({ type, name, ...item }, idx) => {
              if (type === 'quote') {
                const { quote } = item;
                return (
                  <blockquote className="col-span-2 row-span-2 mx-auto my-8 flex max-w-3xl flex-col gap-8 bg-[url(/assets/textures/wave.svg)] bg-contain bg-center bg-no-repeat px-8 sm:col-span-1">
                    <p className="relative text-lg leading-7">
                      <span className="absolute -left-4 top-1 text-2xl leading-3 text-slate-400/80">
                        &ldquo;
                      </span>
                      {quote.text}
                      <span className="ml-1 text-2xl leading-3 text-slate-400/80">&rdquo;</span>
                    </p>
                    <footer className="flex min-w-[180px] flex-row items-center gap-4">
                      <Image
                        src={quote.avatar}
                        alt={`Image of ${quote.attribution.name}`}
                        height="72"
                        width="72"
                        className="h-12 w-12 rounded-full"
                      />
                      <cite className="not-italic leading-8 text-slate-300">
                        <div className="text-base">{quote.attribution.name}</div>
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
                <div className="group relative flex h-full min-h-[148px] items-center rounded-2xl border border-slate-100/10 px-6 py-8">
                  <Image
                    key={idx}
                    src={src}
                    alt={name}
                    title={name}
                    width={120 * scale}
                    height={30 * scale}
                    className={clsx(
                      'width-auto m-auto text-white transition-all',
                      `max-h-[${36 * scale}px]`
                    )}
                  />
                  {!!item.url && (
                    <a
                      href={item.url}
                      target="_blank"
                      className="absolute bottom-2 right-2 hidden flex-row items-center gap-1 rounded-xl  bg-slate-400/10 px-3 py-1.5 text-xs text-slate-50 hover:bg-slate-400/40 group-hover:flex"
                    >
                      Visit website <ArrowTopRightOnSquareIcon className="h-3" />
                    </a>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      </Container>

      <FooterCallout
        title="Talk to a product expert"
        description="Chat with sales engineering to learn how Inngest can help your team ship more reliable products, faster"
        ctaHref="/contact?ref=customers"
        ctaText="Contact sales engineering"
        ctaRef={'customers'}
        showCliCmd={false}
      />

      <Footer disableCta={true} />
    </div>
  );
}
