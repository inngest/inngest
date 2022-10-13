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

  // Only enable redirect URLs to be appended to the appURL to avoid external redirects
  const redirect =
    router.query.redirect?.indexOf("/") === 0
      ? `${appURL}${router.query.redirect}`
      : appURL;

  const handleSubmit = async (e: React.SyntheticEvent) => {
    e.preventDefault();
    setLoading(true);

    let result;
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

      <Header className="header reg-grid section-header">
        <header className="grid-center-6 text-center">
          <h2>Sign up for Inngest Cloud</h2>
          <p>
            The features and functionality that get out of your way and let you
            build.
          </p>
        </header>
      </Header>

      <Content className="grid mx-auto grid-cols-1 md:grid-cols-2 gap-12 px-12 pb-24 max-w-3xl section-header">
        <div className="signup">
          <Button
            href={apiURL(`/v1/login/oauth/github/redirect?anonid=${anonId}`)}
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
            href={apiURL(`/v1/login/oauth/google/redirect?anonid=${anonId}`)}
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

          <form action="" onSubmit={handleSubmit}>
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
    </>
  );
};

const Header = styled.div`
  > div,
  > header {
    padding: 4rem 0;
  }

  header p {
    margin-top: 1rem;
  }
`;

const Content = styled.div`
  .signup {
    display: flex;
    flex-direction: column;

    button + button,
    a + a {
      margin: 0.75rem 0 0 0;
    }
    a img {
      margin: 0 1rem 0 0;
    }
  }

  form,
  .signup > a {
    box-shadow: 0 0 80px rgba(255, 255, 255, 0.06);
  }

  form {
    margin: 2rem 0 0;
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
`;

export default SignUp;
