import React, { useState } from 'react';
import Head from 'next/head'
import styles from '../styles/Home.module.css'
import { INGEST_KEY } from "./index";

const Support = () => {
    const [data, setData] = useState({
        email: "",
        content: "",
    });
    const [error, setError] = useState(null);
    const [buttonText, setButtonText] = useState("Submit");

    const onChange = (field) => (e) => {
        setData(prev => ({
            ...prev, 
            [field]: e.target.value,
        }));
    };

    const isEmailValid = () => {
        // stolen from https://www.w3resource.com/javascript/form/email-validation.php
        return /^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9-]+(?:\.[a-zA-Z0-9-]+)*$/.test(data.email);
      }

    const onSubmit = () => {
        if (!isEmailValid()) {
            setError("Is that a valid email address?");
            return;
        }

        if (!data.content) {
            setError("Please say something :~)");
            return;
        }

        Inngest.init(INGEST_KEY);
        Inngest.event({
            name: "support.request.new",
            data: {
                content: data.content,
            },
            user: {
                email: data.email,
            }
        });

        setButtonText("Done!")
    }

    return <div className={styles.container}>
       <Head>
        <title>Inngest | Support</title>
        <link rel="icon" href="/favicon.png" />
        <meta property="og:title" content="Inngest" />
        <meta property="og:url" content="https://www.inngest.com" />
        <meta property="og:image" content="/logo.svg" />
        <meta property="og:description" content="An event-driven, code automation platform for developers" />
        <script src="/inngest-sdk.js"></script>
      </Head>
      <div className={styles.content}>
      <img style={{ height: '40px'}} className={styles.logo} src="/logo.svg" alt="Inngest logo" />
          <h4>Need help? Talk to us! </h4>
          Join our <a href="https://discord.gg/hUMruzTK">Discord server</a><br/> 
          - or - <br/>
          <input style={{ width: '400px', marginBottom: '20px' }} type="email" placeholder="Your email here" value={data.email} onChange={onChange('email')} />
          <textarea rows={10} style={{ width: '400px', boxSizing: 'border-box', marginBottom: '20px'  }} placeholder="What's up?" value={data.content} onChange={onChange("content")} />
          <button className={styles.submit} onClick={onSubmit}>{buttonText}</button>
          {error && <div style={{ color: 'red', fontSize: '12px', marginTop: '20px'}}>{error}</div>}
      </div>
    </div>
};

export default Support;