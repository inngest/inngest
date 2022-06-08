import styled from "@emotion/styled";
import Nav from "../shared/nav";
import Button from "../shared/Button";
import { useState } from "react";

const api = process.env.REACT_APP_API_HOST || "https://api.inngest.com";

const apiURL = (s: string) => {
  return api + s;
};

const SignUp = () => {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>(null);

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
        body: JSON.stringify({ email, password }),
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
    window.location = "https://app.inngest.com";
  };

  return (
    <>
      <Nav nolinks />

      <Header className="header grid section-header">
        <header className="grid-center-6 text-center">
          <h2>Sign up to Inngest</h2>
          <p>Start building event-driven serverless functions in minutes</p>
        </header>
        <div className="grid-line">
          <span>/01</span>
        </div>
      </Header>

      <Content className="grid section-header">
        <div className="grid-line" />
        <div className="col-1" />
        <div className="signup col-4">
          <Button href={apiURL("/v1/login/oauth/github/redirect")} kind="black">
            <img
              src="https://app.inngest.com/assets/gh-mark.png"
              alt="GitHub"
              width="20"
            />
            <span>
              Sign up with <b>GitHub</b>
            </span>
          </Button>

          <Button href={apiURL("/v1/login/oauth/google/redirect")} kind="black">
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

        <div className="details col-2">
          <h5>Explore Inngest for free</h5>

          <ul className="check">
            <li>Serverless function deploys</li>
            <li>Automatically run functions from events</li>
            <li>Retries & replays built in</li>
            <li>Event schemas, auto-gen'd types, and SDKs</li>
            <li>Unlimited function versions</li>
          </ul>
        </div>

        <div className="grid-line" />
      </Content>
    </>
  );
};

const Header = styled.div`
  > div,
  > header {
    padding: var(--section-padding) 0;
  }

  h2 + p {
    margin: 0;
  }
`;

const Content = styled.div`
  .signup {
    display: flex;
    flex-direction: column;
    padding-bottom: 8vh;

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
    background: rgba(0, 0, 0, 0.4);
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
    padding-left: var(--grid-gap);
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
