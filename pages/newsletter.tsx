import { type GetStaticPropsResult } from "next";
import { useRouter } from "next/router";
import { useRef, useState } from "react";

import Header from "src/shared/Header";
import Logo from "src/shared/Icons/Logo";
import Container from "src/shared/layout/Container";
import Footer from "src/shared/Footer";
import type { PageProps } from "src/shared/types";

export async function getStaticProps(): Promise<
  GetStaticPropsResult<PageProps>
> {
  return {
    props: {
      meta: {
        title: "Newsletter signup",
        description: "Get notified of updates to Inngest",
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
  const tags = tag ? (typeof tag === "string" ? [tag] : tag) : [];

  return (
    <div className="home font-sans bg-slate-1000 bg-x[url(/assets/launch-week/background-image.png)] bg-cover bg-fixed">
      <Header />
      <Container className="py-8 pb-48">
        <div className="my-12 tracking-tight flex items-center justify-center">
          <div className="py-12 md:py-24 rounded-md">
            <h1 className="font-bold text-5xl md:text-7xl leading-tight md:leading-tight text-white text-center mb-4">
              Stay in the loop
            </h1>
            <p className="my-12 text-slate-200 text-lg md:text-xl">
              Be the first to hear about new features, beta releases, and other
              important updates.
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
    error: "",
    result: null,
  });

  const subscribeUser = async (e) => {
    e.preventDefault();
    setLoading(true);
    const res = await fetch("/api/newsletter/subscribe", {
      body: JSON.stringify({
        email: inputRef.current.value,
        tags,
      }),
      headers: {
        "Content-Type": "application/json",
      },
      method: "POST",
    });
    setLoading(false);
    if (res.status === 201) {
      setResponse({ result: true, error: "" });
    } else {
      const { error } = await res.json();
      console.log(error);
      setResponse({ result: false, error });
    }
  };

  const canSubmit = response.result !== true || response.error !== "";

  return (
    <form onSubmit={subscribeUser}>
      <p className="mb-2 text-white text-sm">Get notified:</p>

      <div className="flex flex-row flex-wrap gap-4">
        <input
          className="w-72 flex-grow border border-slate-400 rounded-md px-4 py-2 text-white bg-transparent focus:outline-none focus:ring-1 focus:ring-[#A7F3D0] focus:border-transparent"
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
            className={`whitespace-nowrap button group inline-flex items-center justify-center gap-0.5 rounded-md font-medium tracking-tight transition-all text-white
            bg-indigo-500 hover:bg-indigo-400 text-sm px-3 py-2
            ${loading ? "opacity-40 cursor-not-allowed" : ""}`}
          >
            Register
          </button>
        )}
        <div></div>
      </div>
      {response.error && (
        <p className="mt-2 text-white text-sm">{response.error}</p>
      )}
      {response.result && (
        <p className="mt-2 text-white text-sm">
          Great! You're all set to receive updates on Inngest Launch Week!
        </p>
      )}
    </form>
  );
}
