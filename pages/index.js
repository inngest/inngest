import { useState } from "react";
import Head from 'next/head'
import styles from '../styles/Home.module.css'

const INNGESTION_KEY = 'GCcmd9oe4sAWmS2I6zNx5VZ-LNzAJhKZ7c91ryerqTuu0Ix-Nx2kBbkX9eVA5DS5yu7tfPP9TnbRHs-J69twag';

export default function Home() {
  const [email, setEmail] = useState("");
  const [buttonText, setButtonText] = useState("Submit");

  const onChange = (e) => {
    setEmail(e.target.value);
    setButtonText("Submit");
  }

  const onSubmit = (e) => {
    e.preventDefault();
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
        <form className={styles.form} style={{ marginTop: '10px'}} onSubmit={onSubmit}>
          <input type="email" placeholder="Your email here" value={email} onChange={onChange} />
          <button className={styles.submit} type="submit">{buttonText}</button>
        </form>
        </div>
    </div>
  )
}
