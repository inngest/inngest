import Head from 'next/head';
import Image from 'next/image';
import Link from 'next/link';
import Footer from 'src/shared/Footer';
import FooterCallout from 'src/shared/Footer/FooterCallout';
import Header from 'src/shared/Header';
import Container from 'src/shared/layout/Container';

import { Button } from '../Button';

export type Props = {
  children: React.ReactNode;
  /* The title automatically pulled from the h1 tag in each mdx file */
  title: string;
  /* The optional title used for meta tags (set via export const metaTitle = '...') */
  metaTitle?: string;

  /* Customer info */
  logo: string;
  logoScale?: number;
  companyName: string;
  quote: {
    text: string;
    attribution: {
      name: string;
      title: string;
    };
    avatar: string;
  };
  companyDescription: string;
  companyURL: string;
  ogImage?: string;
};

export function Layout({
  children,
  title,
  logo,
  logoScale = 1,
  companyName,
  quote,
  companyDescription,
  companyURL,
  ogImage = '/assets/customers/case-study/og-image-default.png',
}: Props) {
  const metaTitle = `Case Study - ${companyName}`;
  const description = title;
  return (
    <div className="bg-slate-1000 font-sans">
      <Head>
        <title>{metaTitle}</title>
        <meta name="description" content={description}></meta>
        <meta property="og:title" content={metaTitle} />
        <meta property="og:description" content={description} />
        <meta property="og:image" content={`https://www.inngest.com${ogImage}`} />
        <meta property="og:type" content="article" />
        <meta name="twitter:card" content="summary_large_image" />
        <meta name="twitter:site" content="@inngest" />
        <meta name="twitter:title" content={metaTitle} />
        <meta name="twitter:image" content={`https://www.inngest.com${ogImage}`} />
      </Head>
      <Header />
      <Container>
        <div className="mx-auto my-12 flex max-w-[1200px] flex-col items-start justify-between gap-8 lg:flex-row">
          <div>
            <article className="w-full lg:max-w-[80ch]">
              <div className="mb-4 text-sm font-medium text-slate-500">
                Case Study - {companyName}
              </div>
              <h1 className="mr-8 text-4xl font-medium leading-[3rem]">{title}</h1>

              {quote && (
                <blockquote className="mx-auto my-8 flex max-w-3xl flex-col gap-8 bg-[url(/assets/textures/wave.svg)] bg-[length:auto_80%] bg-center bg-no-repeat px-8 md:flex-row md:p-16">
                  <p className="relative text-lg leading-7">
                    <span className="absolute -left-4 top-1 text-2xl leading-3 text-slate-400/80">
                      &ldquo;
                    </span>
                    {quote.text}
                    <span className="ml-1 text-2xl leading-3 text-slate-400/80">&rdquo;</span>
                  </p>
                  <footer className="flex min-w-[180px] flex-col gap-4">
                    <Image
                      src={quote.avatar}
                      alt={`Image of ${quote.attribution.name}`}
                      height="72"
                      width="72"
                      className="h-12 w-12 rounded-full lg:h-20 lg:w-20"
                    />
                    <cite className="not-italic leading-8 text-slate-300">
                      <span className="text-lg">{quote.attribution.name}</span>
                      <br />
                      <span className="text-sm">{quote.attribution.title}</span>
                    </cite>
                  </footer>
                </blockquote>
              )}

              <div className="prose prose-img:rounded-lg prose-code:bg-slate-800 prose-code:tracking-tight prose-a:text-indigo-400 prose-a:no-underline hover:prose-a:underline hover:prose-a:text-white prose-a:font-medium prose-a:transition-all prose-invert blog-content mb-20 mt-12 text-slate-300 md:max-w-[70ch]">
                {children}
              </div>
            </article>
            <p>
              <Link
                href="/customers"
                className="font-medium text-indigo-400	underline-offset-2 hover:text-slate-50 hover:underline"
              >
                Read more customer success stories →
              </Link>
            </p>
          </div>
          <aside className="top-32 mt-8 flex min-w-[260px] flex-col items-start justify-between gap-6 border-l border-slate-100/10 px-8 py-4 md:sticky md:w-[360px]">
            <img
              src={logo}
              alt={`${companyName}'s logo`}
              style={{ transform: `scale(${logoScale})` }}
              className="mb-4 inline-flex max-h-[40px] min-w-[160px]"
            />
            <p>{companyDescription}</p>
            <p>
              <a href={companyURL} target="_blank" className="text-slate-400 hover:text-indigo-400">
                {companyURL.replace(/https\:\/\//, '')}
              </a>
            </p>
            <div className="mt-2 border-t border-slate-100/10 pt-8">
              <p className="mb-4 font-medium text-slate-50">Interested in Inngest?</p>
              <Button href={`/contact?ref=case-study-${companyName.toLowerCase()}`}>
                Talk to a product expert
              </Button>
            </div>
          </aside>
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
