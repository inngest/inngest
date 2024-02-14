import React from 'react';
import type { GetStaticPropsResult } from 'next';
import Image from 'next/image';
import Link from 'next/link';
import { RocketLaunchIcon } from '@heroicons/react/24/outline';
import Features from 'src/shared/Home/Features';
import Flexibility from 'src/shared/Home/Flexibility';
import Logos from 'src/shared/Home/Logos';
import PlatformFeatures2 from 'src/shared/Home/PlatformFeatures2';
import Quote from 'src/shared/Home/Quote';
import SocialProof from 'src/shared/Home/SocialProof';
import UseCases from 'src/shared/Home/UseCases';
import type { PageProps } from 'src/shared/types';

import Footer from '../shared/Footer';
import Header from '../shared/Header';
import Hero from '../shared/Home/Hero';
import LocalDev from '../shared/Home/LocalDev';
import SocialCTA from '../shared/Home/SocialCTA';
import PageBanner from '../shared/legacy/PageBanner';

export async function getStaticProps(): Promise<GetStaticPropsResult<PageProps>> {
  return {
    props: {
      designVersion: '2',
      meta: {
        title: 'Build reliable products - Durable workflow engine',
        description:
          'Easily develop serverless workflows in your current codebase, without any new infrastructure. Using Inngest, your entire team can ship reliable products.',
        image: '/assets/homepage/open-graph.png',
      },
    },
  };
}

export function LaunchWeekBanner({ urlRef = 'homepage-banner' }) {
  return (
    <PageBanner href={`/launch-week?ref=${urlRef}`} className="mt-px">
      <RocketLaunchIcon className="mr-1 inline-flex h-7 sm:h-5" />
      <span className="shrink">
        We've just wrapped up a Launch Week.{' '}
        <span className="inline-flex font-normal">See all we've shipped.</span>
      </span>
    </PageBanner>
  );
}

export default function Home() {
  return (
    <div
      className="home bg-slate-1000 font-sans"
      style={{
        backgroundImage: `radial-gradient(circle at center -20%, #0e0821 0%, rgba(5, 9, 17, 0) 1500px)`,
        backgroundSize: '100% 1500px',
        backgroundRepeat: 'no-repeat',
      }}
    >
      <Header />

      <LaunchWeekBanner />

      <Hero />

      <Logos
        className="my-20 mb-20 lg:my-36 lg:mb-40 xl:mb-52"
        heading="Helping these teams deliver reliable products"
        logos={[
          {
            src: '/assets/customers/soundcloud-logo-white-horizontal.svg',
            name: 'SoundCloud',
            scale: 1.1,
            featured: true,
          },
          {
            src: '/assets/customers/tripadvisor.svg',
            name: 'TripAdvisor',
            featured: true,
          },
          {
            src: '/assets/customers/resend.svg',
            name: 'Resend',
            scale: 1.1,
          },

          {
            src: '/assets/customers/snaplet-dark.svg',
            name: 'Snaplet',
          },
          {
            src: '/assets/customers/productlane.svg',
            name: 'Productlane',
            scale: 1.3,
          },
          { src: '/assets/customers/leap-logo-white.svg', name: 'Leap' },
        ]}
        footer={
          <div className="flex items-center">
            <Link
              href="/customers"
              className="mx-auto whitespace-nowrap rounded-md border border-slate-800 bg-slate-800 px-6 py-2 font-medium text-white transition-all hover:border-slate-600 hover:bg-slate-500/10 hover:bg-slate-600"
            >
              Read customer case studies
            </Link>
          </div>
        }
      />

      <UseCases />

      <div className="my-32 lg:my-48">
        <Quote
          text={`Inngest is an essential partner to Vercel's frontend cloud offering. It extends Vercel's DX and serverless operational model to a notoriously challenging yet crucial part of the stack: backend workflows and asynchronous process coordination.`}
          attribution={{
            name: 'Guillermo Rauch',
            title: (
              <span>
                <span className="text-white">▲</span> CEO of Vercel
              </span>
            ),
            avatar: '/assets/about/guillermo-rauch-avatar.jpg',
          }}
        />
      </div>

      <Features />

      <div className="my-32">
        <Quote
          text="The DX and visibility with Inngest is really incredible. We are able to develop functions locally easier and faster that with our previous queue. Also, Inngest's tools give us the visibility to debug issues much quicker than before."
          attribution={{
            name: 'Bu Kinoshita',
            title: 'Co-founder @ Resend',
            logo: '/assets/customers/resend.svg',
          }}
          caseStudy="/customers/resend?ref=homepage"
        />
      </div>

      <LocalDev className="mb-24" />

      <div className="mb-48 mt-32">
        <Quote
          text="The DX and code simplicity it brings is unmatched, especially around local development. We're currently working to migrate some of our larger systems over and it’s a joy to see all the complexity it replaces, and with a much better story around partial failures and retries."
          attribution={{
            name: 'Justin Cypret',
            title: 'Director of Engineer @ Zamp',
            logo: '/assets/customers/zamp-logo.svg',
          }}
        />
      </div>

      <PlatformFeatures2 />

      <div className="my-32">
        <Quote
          text="We switched from our PostgreSQL backed queue to Inngest in less than a day. Their approach is idiomatic with a great developer experience. Inngest allowed us to stop worrying about scalability and stability."
          attribution={{
            name: 'Peter Pistorius',
            title: 'CEO @ Snaplet',
            avatar: '/assets/customers/snaplet-peter-pistorius.png',
          }}
        />
      </div>

      <Flexibility />

      {/* <EnterpriseTrust /> */}

      <div className="my-32">
        <Quote
          text={`I can't stress enough how integral Inngest has been to our operations. It's more than just "battle tested" for us—it's been a game-changer and a cornerstone of our processes.`}
          attribution={{
            name: 'Robin Curbelo',
            title: 'Engineer @ Niftykit',
            avatar: '/assets/customers/niftykit-robin-curbelo.jpg',
          }}
        />
      </div>

      <SocialProof />

      <SocialCTA />

      <Footer />
    </div>
  );
}
