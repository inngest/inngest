import { useRouter } from "next/router";
import styled from "@emotion/styled";
import Logo from "../shared/Icons/Logo";
import { useState } from "react";
import { useAnonymousID } from "src/shared/legacy/trackingHooks";

export async function getStaticProps() {
  return {
    props: {
      designVersion: "2",
      meta: {
        title: "Sign up to Inngest",
      },
    },
  };
}

const api = process.env.NEXT_PUBLIC_API_HOST || "https://api.inngest.com";
const appURL = process.env.NEXT_PUBLIC_APP_HOST || "https://app.inngest.com";

const apiURL = (s: string) => {
  return api + s;
};

const SignUp = () => {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>(null);
  const { anonymousID } = useAnonymousID();
  const router = useRouter();

  const loc = router.query.to || "/";
  const to = loc.indexOf("/") === 0 ? `${appURL}${loc}` : loc;

  const b64search = router.query.search || "";
  // search is the base64-encoded search string that the previous authenticated
  // page had.  As we replace the query string with "to", we store this entire
  // string base64 encoded so that we can drop this on the new page after login.
  // @ts-ignore
  const search = atob(b64search);
  const redirect = `${to}${search}`;

  const handleSubmit = async (e: React.SyntheticEvent) => {
    e.preventDefault();
    setLoading(true);

    let result: any;
    try {
      result = await fetch(apiURL(`/v1/register`), {
        method: "POST",
        credentials: "include",
        headers: {
          "content-type": "application/json",
        },
        body: JSON.stringify({ email, password, anon_id: anonymousID }),
      });
    } catch (e) {
      setError("There was an error signing you up.  Please try again.");
      setLoading(false);
      return;
    }

    if (result.status !== 200) {
      setLoading(false);
      try {
        const json = await result.json();
        setError(json.error);
      } catch (e) {
        setError("There was an error signing you up.  Please try again.");
      }
      return;
    }

    setLoading(false);
    // @ts-ignore
    window.location = redirect;
  };

  return (
    <Wrapper className="flex h-screen w-screen">
      <div className="flex h-full w-full items-center justify-items-center overflow-y-scroll bg-white px-4 sm:w-2/3 md:w-1/2">
        <div className="mx-auto my-8 w-80 text-center">
          <Logo className="mb-8 flex justify-center" />
          <h2 className="text-slate-800 font-semibold">Sign up for Inngest Cloud</h2>
          <div className="my-5 flex flex-col gap-4 text-center">
            <a
              className="inline-flex flex-shrink-0 items-center gap-1 justify-center overflow-hidden text-sm font-regular rounded-[6px] transition-all shadow-outline-secondary-light font-medium px-6 py-2.5 bg-gray-900 text-white shadow-sm hover:bg-gray-700 hover:text-white"
              href={apiURL(
                `/v1/login/oauth/github/redirect?anonid=${anonymousID}&to=${loc}&search=${b64search}`
              )}
            >
              <img
                src="/assets/ui-assets/source-logos/github-mark-white.png"
                alt="GitHub"
                className="mr-2 w-4 h-4"
              />
              <span>
                Sign up with <b>GitHub</b>
              </span>
            </a>

            <a
              className="inline-flex flex-shrink-0 items-center gap-1 justify-center overflow-hidden text-sm font-regular rounded-[6px] transition-all shadow-outline-secondary-light font-medium px-6 py-2.5 bg-slate-700 text-white shadow-sm hover:bg-slate-500 hover:text-white"
              href={apiURL(
                `/v1/login/oauth/google/redirect?anonid=${anonymousID}&to=${loc}&search=${b64search}`
              )}
              style={{ marginLeft: 0 }}
            >
              <img
                src="/assets/ui-assets/icons/google.svg"
                alt="Google"
                className="mr-2 w-4 h-4"
              />
              <span>
                Sign up with <b>Google</b>
              </span>
            </a>
          </div>
          <form
            className="flex flex-col gap-4 border-b border-t border-slate-300 text-black pb-6 pt-4 text-center"
            action=""
            onSubmit={handleSubmit}
          >
            <p className="text-center text-sm font-medium text-slate-600">
              Or sign up with email
            </p>
            {error && <p className="text-red-800">{error}</p>}
            <div className="flex flex-col gap-1">
              <input
                type="email"
                name="email"
                placeholder="Email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="border border-slate-300 placeholder-slate-500 shadow outline-2 outline-offset-2 outline-indigo-500 transition-all focus:outline text-sm px-3.5 py-3 rounded-lg"
              />
            </div>
            <div className="flex flex-col gap-1">
              <input
                type="password"
                name="password"
                placeholder="Password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="border border-slate-300 placeholder-slate-500 shadow outline-2 outline-offset-2 outline-indigo-500 transition-all focus:outline text-sm px-3.5 py-3 rounded-lg"
              />
            </div>

            <button
              className="inline-flex flex-shrink-0 items-center gap-1 justify-center overflow-hidden text-sm font-regular rounded-[6px] transition-all bg-gradient-to-b from-[#6d7bfe] to-[#6366f1] hover:from-[#7986fd] hover:to-[#7679f9] text-shadow text-white font-medium px-6 py-2.5"
              type="submit"
              disabled={loading}
            >
              Sign up via email
            </button>
          </form>
          <p className="mt-4 text-center text-sm font-medium text-slate-700">
            Do you already have an account?{" "}
            <a
              className="underline"
              href="https://app.inngest.com/login?ref=signup"
            >
              Log in here
            </a>
          </p>
        </div>
      </div>
      <div className="mesh-gradient hidden w-1/3 sm:block md:w-1/2" />
    </Wrapper>
  );
};

const Wrapper = styled.div`
  .mesh-gradient {
    background-color: hsla(240, 100%, 72%, 1);
    background-image: radial-gradient(
        at 62% 96%,
        hsla(191, 100%, 59%, 0.85) 0px,
        transparent 50%
      ),
      radial-gradient(at 48% 9%, hsla(275, 100%, 69%, 1) 0px, transparent 50%),
      radial-gradient(at 8% 11%, hsla(240, 100%, 69%, 1) 0px, transparent 50%),
      radial-gradient(at 3% 93%, hsla(0, 78%, 70%, 0.6) 0px, transparent 50%),
      radial-gradient(at 89% 26%, hsla(266, 100%, 57%, 1) 0px, transparent 50%),
      radial-gradient(at 84% 79%, hsla(240, 100%, 70%, 1) 0px, transparent 50%);
  }
`;

export default SignUp;
