import { useRef, useState } from 'react';
import { type GetStaticPropsResult } from 'next';
import { useRouter } from 'next/router';
import Footer from 'src/shared/Footer';
import Header from 'src/shared/Header';
import Logo from 'src/shared/Icons/Logo';
import Container from 'src/shared/layout/Container';
import type { PageProps } from 'src/shared/types';

export async function getStaticProps(): Promise<GetStaticPropsResult<PageProps>> {
  return {
    props: {
      meta: {
        title: 'Newsletter signup',
        description: 'Get notified of updates to Inngest',
      },
    },
  };
}

// Note add ?tag=<mailchimp-tag> to tag the user with the interest
// ex. /newsletter?tag=sdk-go
export default function LaunchWeek() {
  const router = useRouter();
  const { tag } = router.query;
  // Can be multiple tags
  const tags = tag ? (typeof tag === 'string' ? [tag] : tag) : [];

  return (
    <div className="home bg-slate-1000 bg-x[url(/assets/launch-week/background-image.png)] bg-cover bg-fixed font-sans">
      <Header />
      <Container className="py-8 pb-48">
        <div className="my-12 flex items-center justify-center tracking-tight">
          <div className="rounded-md py-12 md:py-24">
            <h1 className="mb-4 text-center text-5xl font-bold leading-tight text-white md:text-7xl md:leading-tight">
              Stay in the loop
            </h1>
            <p className="my-12 text-lg text-slate-200 md:text-xl">
              Be the first to hear about new features, beta releases, and other important updates.
            </p>

            <NewsletterSignup tags={tags} />
          </div>
        </div>
      </Container>

      <Footer disableCta={true} />
    </div>
  );
}

function NewsletterSignup({ tags = [] }: { tags: string[] }) {
  const inputRef = useRef(null);
  const [loading, setLoading] = useState(false);
  const [response, setResponse] = useState<{
    error: string;
    result: boolean | null;
  }>({
    error: '',
    result: null,
  });

  const subscribeUser = async (e) => {
    e.preventDefault();
    setLoading(true);
    const res = await fetch('/api/newsletter/subscribe', {
      body: JSON.stringify({
        email: inputRef.current.value,
        tags,
      }),
      headers: {
        'Content-Type': 'application/json',
      },
      method: 'POST',
    });
    setLoading(false);
    if (res.status === 201) {
      setResponse({ result: true, error: '' });
    } else {
      const { error } = await res.json();
      console.log(error);
      setResponse({ result: false, error });
    }
  };

  const canSubmit = response.result !== true || response.error !== '';

  return (
    <form onSubmit={subscribeUser}>
      <p className="mb-2 text-sm text-white">Get notified:</p>

      <div className="flex flex-row flex-wrap gap-4">
        <input
          className="w-72 flex-grow rounded-md border border-slate-400 bg-transparent px-4 py-2 text-white focus:border-transparent focus:outline-none focus:ring-1 focus:ring-[#A7F3D0]"
          type="email"
          id="email-input"
          name="email"
          placeholder="Enter your email address"
          ref={inputRef}
          required
          autoCapitalize="off"
          autoCorrect="off"
        />
        {canSubmit && (
          <button
            type="submit"
            name="register"
            disabled={loading || response.result === true}
            className={`button group inline-flex items-center justify-center gap-0.5 whitespace-nowrap rounded-md bg-indigo-500 px-3 py-2 text-sm
            font-medium tracking-tight text-white transition-all hover:bg-indigo-400
            ${loading ? 'cursor-not-allowed opacity-40' : ''}`}
          >
            Register
          </button>
        )}
        <div></div>
      </div>
      {response.error && <p className="mt-2 text-sm text-white">{response.error}</p>}
      {response.result && (
        <p className="mt-2 text-sm text-white">
          Great! You're all set to receive updates on Inngest Launch Week!
        </p>
      )}
    </form>
  );
}
