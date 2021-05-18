import { useState } from "react";
import Head from 'next/head'
import styles from '../styles/Home.module.css'

const INNGESTION_KEY = 'BIjxBrM6URqxAu0XgIAae5HgBCv8l_LodmdGonFCfngjhwIgQEbvbUUQTwvFMHO21vxCJEGsC7KPdXEzdXgOAQ';

export default function Home() {
  const [email, setEmail] = useState("");
  const [buttonText, setButtonText] = useState("Submit");
  const [lastSubmitted, setLastSubmitted] = useState(null);
  const [error, setError] = useState(null);

  const onChange = (e) => {
    setEmail(e.target.value);
    setButtonText("Submit");
  }

  const isEmailValid = () => {
    // stolen from https://www.w3resource.com/javascript/form/email-validation.php
    return /^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9-]+(?:\.[a-zA-Z0-9-]+)*$/.test(email);
  }

  const onSubmit = (e) => {
    e.preventDefault();

    if (!isEmailValid()) {
      setError("Is that a valid email address?");
      return;
    };

    // prevent users from spamming the submit button
    if (lastSubmitted === email) {
      return;
    }

    setError(null);
    setLastSubmitted(email);

    const Inngest = globalThis.Inngest;

    if (!Inngest) return;

    Inngest.init(INNGESTION_KEY);
    Inngest.event({
      name: "marketing.signup",
      data: {
        email,
      },
      user: {
        email,
      }
    });

    setButtonText("Done!");
  }

  return (
    <div className={styles.container}>
      <Head>
        <title>Inngest</title>
        <link rel="icon" href="/favicon.png" />
        <meta property="og:title" content="Inngest" />
        <meta property="og:url" content="https://www.inngest.com" />
        <meta property="og:image" content="/logo.svg" />
        <meta property="og:description" content="An event-driven, code automation platform for developers" />
        <script src="/inngest-sdk.js"></script>
      </Head>

      <div className={styles.content}>
        <img className={styles.logo} src="/logo.svg" alt="Inngest logo" />
        <div><b>the first event-driven, code automation platform for developers</b></div>
        <br/>
        <div>
          event subscriptions <br /> 
          event coordination <br /> 
          scheduling <br /> 
          DAG workflows <br/>
          workflow functions-as-a-service<br/>
        </div>
        <br/>
        <b>Sign up for updates</b>
        <div>
          <input type="email" placeholder="Your email here" value={email} onChange={onChange} />
          <button disabled={email === lastSubmitted} className={styles.submit} onClick={onSubmit}>{buttonText}</button>
        </div>
        {error && <div style={{ color: 'red', fontSize: '12px', marginTop: "5px" }}>{error}</div>}
        </div>
    </div>
  )
}
