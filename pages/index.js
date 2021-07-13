import { useState } from "react";
import styled from "@emotion/styled";
import Head from 'next/head'
import styles from '../styles/Home.module.css'

// TODO: move these into env vars 
// prod key
export const INGEST_KEY = 'BIjxBrM6URqxAu0XgIAae5HgBCv8l_LodmdGonFCfngjhwIgQEbvbUUQTwvFMHO21vxCJEGsC7KPdXEzdXgOAQ';
 
// test key
// export const INGEST_KEY = 'MnzaTCk7Se8i74hA141bZGS-NY9P39RSzYFbxanIHyV2VDNu1fwrns2xBQCEGdIb9XRPtzbp0zdRPjtnA1APTQ';

const Check = ({ size = 16, color = "#5ea659" }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
<path d="M12 22C6.47715 22 2 17.5228 2 12C2 6.47715 6.47715 2 12 2C17.5228 2 22 6.47715 22 12C21.9939 17.5203 17.5203 21.9939 12 22ZM11.984 20H12C16.4167 19.9956 19.9942 16.4127 19.992 11.996C19.9898 7.57929 16.4087 4 11.992 4C7.57528 4 3.99421 7.57929 3.992 11.996C3.98979 16.4127 7.56729 19.9956 11.984 20ZM10 17L6 13L7.41 11.59L10 14.17L16.59 7.58L18 9L10 17Z" fill={color}></path>
</svg>
)

export default function Home() {

  return (
    <>
      <Nav>
        <div>
          <img src="/logo-blue.svg" alt="Inngest logo" />
        </div>
        <div>
          <a href="https://app.inngest.com/login">Sign in</a>
          <a href="#" className="button">Request a free demo</a>
        </div>
      </Nav>

      <Hero className="text-center">
        <h1>Trigger low-code logic from events</h1>
        <p>
          Companies use Inngest to build real time, event driven workflows in minutes. <br />
          It's <u>made for builders</u>, <u>designed for operators</u>.
        </p>

        <div>
          <img src="/wflow.png" alt="An example cloud kitchen workflow when paying via Venmo" />
        </div>
      </Hero>


      <Content>
        <Tagline>
          <div>
            <Check />
            <p>Define workflows as code or&nbsp;via&nbsp;a&nbsp;UI</p>
          </div>
          <div>
            <Check />
            <p>Utilize pre-built integrations</p>
          </div>
          <div>
            <Check />
            <p>Run your own code, using&nbsp;any&nbsp;language</p>
          </div>
          <div>
            <Check />
            <p>Full user transparency &amp; audit&nbsp;trails</p>
          </div>
          <div>
            <Check />
            <p>Complete workflow version histories</p>
          </div>
        </Tagline>
      </Content>

      <HIW>
        <Content>
          <h5>Introducing Inngest</h5>

          <p>Inngest is an <strong>automation platform</strong> which <strong>runs workflows on a schedule</strong> or <strong>in real-time after events happen</strong>. Design&nbsp;<strong>complex operational flows</strong> and <strong>run any code</strong> - including pre-built integrations or your own code - with <strong>zero&nbsp;infrastructure&nbsp;and&nbsp;maintenance</strong>.</p>

          <HIWGrid>
            <div>
              <h2>Workflow automation</h2>
              <p>Build, manage, and operate your product and ops flows end-to-end.  Complete with out-of-the-box integrations for rapid development, and the ability to run your own serverless code for full&nbsp;flexibility</p>
            </div>
            <div>
              <h2>Change management</h2>
              <p>Version every workflow complete with history, schedule workflows to go live, and handle workflow approvals within your account - it’s everything you need for a fully compliant&nbsp;solution</p>
            </div>
            <div>
              <h2>Transparency &amp; debugging</h2>
              <p>Drill down into every workflow run, including which users ran through which versions of a workflow and each workflow’s&nbsp;logs.</p>
            </div>
          </HIWGrid>
        </Content>
      </HIW>

      <Content>
        <Callout className="text-center">
          <div>
            <span>35x</span>
            <strong>faster implementation</strong>
            <span>using our platform and integrations</span>
          </div>

          <div>
            <span>20x</span>
            <strong>faster debugging &amp; editing</strong>
            <span>with our insights, logs, and editor</span>
          </div>

          <div>
            <span>15x</span>
            <strong>more cost effective</strong>
            <span>than deploying &amp; managing yourself</span>
          </div>
        </Callout>
      </Content>

      <Footer />
    </>
  )
}

const Content = styled.div`
  max-width: 1200px;
  margin: 0 auto;
`

const Nav = styled(Content)`
  height: 70px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  font-size: .9rem;

  img {
    max-height: 40px;
  }

  a:not(.button) {
    display: inline-block;
    padding: 12px 20px 11px;
    transition: all .2s;
  }
  a:not(.button):hover {
    background: #2F6D9D11;
  }

  a + a {
  margin-left: 20px;
  }
`


const Hero = styled(Content)`
  font-size: 1.3125rem;
  padding: 80px 0 0;
  position: relative;

  > div {
    box-shadow: 0 10px 50px rgba(0, 0, 0, 0.1);
    background: #FFFEFC;
    width: 100%;
    max-width: 100%;
    height: 480px;
    max-height: 500px;
    margin: 100px 0 0;
    position: relative;
    z-index: 2;
    overflow: hidden;
  }
`

const Tagline = styled.div`
  padding: 40px 40px 0;
  font-size: 12px;
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  grid-gap: 40px;
  text-align: center;

  > div {
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  p {
    margin: 4px 0;
    line-height: 1.3;
    opacity: .65;
  }
`;

const HIW = styled.div`
  box-shadow: inset 0 0 0 20px #fff;
  background: linear-gradient(180deg, rgba(243,245,245,1) 20%, rgba(249,251,254,1) 100%);;
  padding: 450px 40px 180px 40px;
  margin-top: -400px;

  h5 + p {
    font-size: 1.3125rem;
  }
`

const HIWGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  grid-gap: 100px;
  padding: 30px 0 0;
`

const Callout = styled.div`
  max-width: 80%;
  margin: -115px auto 0 auto;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  grid-gap: 40px;

  background: #FDFBF6;
  padding: 40px;
  box-shadow: 0 10px 50px rgba(0, 0, 0, 0.1);

  strong, span {
    display: block;
    margin: 4px 0;
  }

  span:first-of-type {
    font-size: 2.6rem;
    margin: 0 0 6px;
  }

  span:last-of-type {
    color: #737885;
  }
`;

const Footer = styled.div`
  margin-top: 100px;
`
