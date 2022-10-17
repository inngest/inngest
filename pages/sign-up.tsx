import { useRouter } from "next/router";
import styled from "@emotion/styled";
import Nav from "../shared/nav";
import Button from "../shared/Button";
import IconList from "../shared/IconList";
import Check from "../shared/Icons/Check";
import { useState } from "react";
import { useAnonId } from "src/shared/trackingHooks";

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
  const { anonId } = useAnonId();
  const router = useRouter();

  const to = router.query.to || "/";
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
        body: JSON.stringify({ email, password, anon_id: anonId }),
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
    <>
      <Nav nolinks />

      <div className="mx-auto max-w-3xl px-8">
        <Header className="py-16 text-center">
          <h2>Sign up for Inngest Cloud</h2>
          <p className="mt-4">
            The features and functionality that get out of your way and let you
            build.
          </p>
        </Header>
        <PopUpHeader className="mb-6 text-center">
          <h2 className="text-xl">
            Sign up for Inngest Cloud {to !== "/" && "to continue"}
          </h2>
        </PopUpHeader>

        <Content className="grid gap-12">
          <div className="signup">
            <div className="buttons">
              <Button
                href={apiURL(
                  `/v1/login/oauth/github/redirect?anonid=${anonId}&to=${to}&search=${b64search}`
                )}
                kind="black"
              >
                <img
                  src="https://app.inngest.com/assets/gh-mark.png"
                  alt="GitHub"
                  width="20"
                />
                <span>
                  Sign up with <b>GitHub</b>
                </span>
              </Button>

              <Button
                href={apiURL(
                  `/v1/login/oauth/google/redirect?anonid=${anonId}&to=${to}&search=${b64search}`
                )}
                kind="black"
                style={{ marginLeft: 0 }}
              >
                <img
                  src="https://app.inngest.com/assets/icons/google.svg"
                  alt="Google"
                  width="20"
                />
                <span>
                  Sign up with <b>Google</b>
                </span>
              </Button>
            </div>

            <form className="form" action="" onSubmit={handleSubmit}>
              <label>
                Email
                <input
                  type="email"
                  name="email"
                  placeholder="you@company.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                />
              </label>
              <label>
                Password
                <input
                  type="password"
                  name="password"
                  placeholder="Create a strong password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                />
              </label>

              {error && <p className="error">{error}</p>}

              <Button type="submit" kind="primary" disabled={loading}>
                Sign up via email
              </Button>
            </form>
          </div>

          <div className="details">
            <h5>Try Inngest for free</h5>
            <IconList
              direction="vertical"
              items={[
                "No queues to configure",
                "Full event history",
                "Serverless functions",
                "Any programming language",
                "Step function support with DAGs",
                "Bring your whole team",
              ].map((text) => ({
                icon: Check,
                text,
              }))}
            />
          </div>
        </Content>
      </div>
    </>
  );
};

const Header = styled.header`
  // Hide for short pop-up windows, e.g. Vercel integration flow (e.g. 800x600)
  @media (max-height: 800px) and (min-width: 640px) {
    display: none;
  }
`;
const PopUpHeader = styled.div`
  display: none;
  @media (max-height: 800px) and (min-width: 640px) {
    display: block;
  }
`;

const Content = styled.div`
  grid-template-columns: repeat(2, 1fr);

  .signup {
    display: grid;
    gap: 2rem;
  }

  .buttons {
    display: flex;
    flex-direction: column;
    justify-content: center;

    button + button,
    a + a {
      margin: 0.75rem 0 0 0;
    }
    a img {
      margin: 0 1rem 0 0;
    }
  }

  .form,
  .buttons > a {
    box-shadow: 0 0 80px rgba(255, 255, 255, 0.06);
  }

  .form {
    padding: 2rem;
    border-radius: var(--border-radius);
    background: linear-gradient(
      135deg,
      hsl(332deg 30% 95%) 0%,
      hsl(240deg 30% 95%) 100%
    );
    display: flex;
    flex-direction: column;
  }

  label {
    font-size: 0.85rem;
    display: flex;
    flex-direction: column;
  }
  label + label {
    margin: 1.25rem 0 0;
  }
  input {
    margin: 0.2rem 0 0;
  }
  button {
    margin: 2.5rem 0 0;
  }

  .error {
    text-align: center;
    margin: 2rem 0 -1rem;
    font-size: 0.9rem;
    font-weight: 500;
    color: red;
  }

  .details {
    opacity: 0.85;
    font-size: 0.85rem;
    display: flex;
    flex-direction: column;
    justify-content: center;

    ul {
      margin: 2rem 0;
    }
  }

  // mobile
  @media (max-width: 768px) {
    grid-template-columns: 1fr;
  }

  // Hide for short pop-up windows, e.g. Vercel integration flow (e.g. 800x600)
  @media (max-height: 800px) and (min-width: 640px) {
    grid-template-columns: 1fr;
    .signup {
      display: grid;
      grid-template-columns: 1fr 1fr;
    }
    .details {
      display: none;
    }
  }
`;

export default SignUp;
