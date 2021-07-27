import { useState } from "react";
import Head from "next/head";
import styles from "../styles/Home.module.css";

// TODO: move these into env vars
// prod key
export const INGEST_KEY =
  "BIjxBrM6URqxAu0XgIAae5HgBCv8l_LodmdGonFCfngjhwIgQEbvbUUQTwvFMHO21vxCJEGsC7KPdXEzdXgOAQ";

// test key
// export const INGEST_KEY = 'MnzaTCk7Se8i74hA141bZGS-NY9P39RSzYFbxanIHyV2VDNu1fwrns2xBQCEGdIb9XRPtzbp0zdRPjtnA1APTQ';

export default function Home() {
  const [email, setEmail] = useState("");
  const [buttonText, setButtonText] = useState("Submit");
  const [lastSubmitted, setLastSubmitted] = useState(null);
  const [error, setError] = useState(null);

  const onChange = (e) => {
    setEmail(e.target.value);
    setError(null);
    setButtonText("Submit");
  };

  const isEmailValid = () => {
    // stolen from https://www.w3resource.com/javascript/form/email-validation.php
    return /^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9-]+(?:\.[a-zA-Z0-9-]+)*$/.test(
      email
    );
  };

  const onSubmit = () => {
    if (!isEmailValid()) {
      setError("Is that a valid email address?");
      return;
    }

    // prevent users from spamming the submit button
    if (lastSubmitted === email) {
      return;
    }

    setError(null);
    setLastSubmitted(email);

    const Inngest = globalThis.Inngest;

    if (!Inngest) return;

    Inngest.init(INGEST_KEY);
    Inngest.event({
      name: "marketing.signup",
      data: {
        email,
      },
      user: {
        email,
      },
    });

    setButtonText("Done!");
  };

  const onInputKey = (e) => {
    // enter key
    if (e.keyCode === 13) {
      onSubmit();
    }
  };

  return (
    <div className={styles.container}>
      <Head>
        <title>Inngest</title>
        <link rel="icon" href="/favicon.png" />
        <meta property="og:title" content="Inngest" />
        <meta property="og:url" content="https://www.inngest.com" />
        <meta property="og:image" content="/logo.svg" />
        <meta
          property="og:description"
          content="An event-driven code automation platform for developers"
        />
        <script src="/inngest-sdk.js"></script>
      </Head>

      <div className={styles.content}>
        <img className={styles.logo} src="/logo.svg" alt="Inngest logo" />

        <hgroup>
          <h1>
            <b>serverless event driven infrastructure</b> for engineers
          </h1>
          <h2>Build and deploy 25x faster on Inngest</h2>
        </hgroup>

        <div>
          event subscriptions <br />
          event coordination <br />
          workflow functions-as-a-service, solo or as a DAG
          <br />
          scheduled DAG workflows <br />
        </div>

        <br />

        <b>Sign up for updates</b>
        <br />

        <div>
          <input
            onKeyDown={onInputKey}
            type="email"
            placeholder="Your email here"
            value={email}
            onChange={onChange}
          />
          <button
            disabled={email === lastSubmitted}
            className={styles.submit}
            onClick={onSubmit}
          >
            {buttonText}
          </button>
        </div>
        {error && (
          <div style={{ color: "red", fontSize: "12px", marginTop: "5px" }}>
            {error}
          </div>
        )}
      </div>
    </div>
  );
}
