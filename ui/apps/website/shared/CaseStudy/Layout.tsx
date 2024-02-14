import Head from "next/head";
import Link from "next/link";
import Image from "next/image";

import Header from "src/shared/Header";
import Container from "src/shared/layout/Container";
import Footer from "src/shared/Footer";
import FooterCallout from "src/shared/Footer/FooterCallout";
import { Button } from "../Button";

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
  ogImage = "/assets/customers/case-study/og-image-default.png",
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
        <meta
          property="og:image"
          content={`https://www.inngest.com${ogImage}`}
        />
        <meta property="og:type" content="article" />
        <meta name="twitter:card" content="summary_large_image" />
        <meta name="twitter:site" content="@inngest" />
        <meta name="twitter:title" content={metaTitle} />
        <meta
          name="twitter:image"
          content={`https://www.inngest.com${ogImage}`}
        />
      </Head>
      <Header />
      <Container>
        <div className="mx-auto my-12 flex flex-col lg:flex-row items-start justify-between gap-8 max-w-[1200px]">
          <div>
            <article className="w-full lg:max-w-[80ch]">
              <div className="mb-4 text-sm font-medium text-slate-500">
                Case Study - {companyName}
              </div>
              <h1 className="mr-8 text-4xl leading-[3rem] font-medium">
                {title}
              </h1>

              {quote && (
                <blockquote className="mx-auto my-8 max-w-3xl px-8 md:p-16 flex flex-col md:flex-row gap-8 bg-[url(/assets/textures/wave.svg)] bg-[length:auto_80%] bg-center bg-no-repeat">
                  <p className="text-lg leading-7 relative">
                    <span className="absolute top-1 -left-4 text-2xl leading-3 text-slate-400/80">
                      &ldquo;
                    </span>
                    {quote.text}
                    <span className="ml-1 text-2xl leading-3 text-slate-400/80">
                      &rdquo;
                    </span>
                  </p>
                  <footer className="min-w-[180px] flex flex-col gap-4">
                    <Image
                      src={quote.avatar}
                      alt={`Image of ${quote.attribution.name}`}
                      height="72"
                      width="72"
                      className="rounded-full h-12 w-12 lg:h-20 lg:w-20"
                    />
                    <cite className="text-slate-300 leading-8 not-italic">
                      <span className="text-lg">{quote.attribution.name}</span>
                      <br />
                      <span className="text-sm">{quote.attribution.title}</span>
                    </cite>
                  </footer>
                </blockquote>
              )}

              <div className="md:max-w-[70ch] prose mt-12 mb-20 prose-img:rounded-lg prose-code:bg-slate-800 prose-code:tracking-tight text-slate-300 prose-a:text-indigo-400 prose-a:no-underline hover:prose-a:underline hover:prose-a:text-white prose-a:font-medium prose-a:transition-all prose-invert blog-content">
                {children}
              </div>
            </article>
            <p>
              <Link
                href="/customers"
                className="hover:underline underline-offset-2	text-indigo-400 hover:text-slate-50 font-medium"
              >
                Read more customer success stories →
              </Link>
            </p>
          </div>
          <aside className="md:sticky top-32 flex flex-col gap-6 min-w-[260px] md:w-[360px] px-8 py-4 mt-8 items-start justify-between border-l border-slate-100/10">
            <img
              src={logo}
              alt={`${companyName}'s logo`}
              style={{ transform: `scale(${logoScale})` }}
              className="inline-flex min-w-[160px] max-h-[40px] mb-4"
            />
            <p>{companyDescription}</p>
            <p>
              <a
                href={companyURL}
                target="_blank"
                className="text-slate-400 hover:text-indigo-400"
              >
                {companyURL.replace(/https\:\/\//, "")}
              </a>
            </p>
            <div className="mt-2 pt-8 border-t border-slate-100/10">
              <p className="mb-4 font-medium text-slate-50">
                Interested in Inngest?
              </p>
              <Button
                href={`/contact?ref=case-study-${companyName.toLowerCase()}`}
              >
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
        ctaRef={"customers"}
        showCliCmd={false}
      />

      <Footer disableCta={true} />
    </div>
  );
}
