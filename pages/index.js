import { useState } from "react";
import styled from "@emotion/styled";
import Head from 'next/head'
import styles from '../styles/Home.module.css'

// TODO: move these into env vars 
// prod key
export const INGEST_KEY = 'BIjxBrM6URqxAu0XgIAae5HgBCv8l_LodmdGonFCfngjhwIgQEbvbUUQTwvFMHO21vxCJEGsC7KPdXEzdXgOAQ';
 
// test key
// export const INGEST_KEY = 'MnzaTCk7Se8i74hA141bZGS-NY9P39RSzYFbxanIHyV2VDNu1fwrns2xBQCEGdIb9XRPtzbp0zdRPjtnA1APTQ';

export default function Home() {

  return (
    <>
      <Nav>
        <img src="/logo-blue.svg" alt="Inngest logo" />
      </Nav>

      <Hero className="text-center">
        <h1>Automate your workflows</h1>
        <p>
          Build real time, event driven workflows in minutes with our serverless platform. <br />
          Define workflows as code or via a UI, utilize pre-built integrations, or run your own code:<br />
          it's <u>made for builders</u>, <u>designed for operators</u>.
        </p>

        <div />
      </Hero>

      <HIW>
        <Content>
          <h5>Introducing Inngest</h5>

          <p>Inngest is an <strong>automation platform</strong> which <strong>runs workflows on a schedule</strong> or <strong>in real-time after events happen</strong>. Design&nbsp;<strong>complex operational flows</strong> and <strong>run any code</strong> - including pre-built integrations or your own code - with <strong>zero&nbsp;infrastructure&nbsp;and&nbsp;maintenance</strong>.</p>

          <HIWGrid>
            <div>
              <h2>Workflow management</h2>
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
            <span>35x</span>
            <strong>faster implementation</strong>
            <span>using our platform and integrations</span>
          </div>

          <div>
            <span>35x</span>
            <strong>faster implementation</strong>
            <span>using our platform and integrations</span>
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
  padding: 0 20px;

  img {
    max-height: 40px;
  }
`


const Hero = styled(Content)`
  font-size: 1.3125rem;
  padding: 80px 0 0;
  position: relative;

  > div {
    box-shadow: 0 10px 50px rgba(0, 0, 0, 0.1);
    background: #FDFBF6;
    width: 100%;
    height: 500px;
    margin: 100px 0 0;
    position: relative;
    z-index: 2;
  }
`

const HIW = styled.div`
  box-shadow: inset 0 0 0 20px #fff;
  background: linear-gradient(180deg, rgba(243,245,245,1) 20%, rgba(249,251,254,1) 100%);;
  padding: 450px 40px 140px 40px;
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
  margin: -80px auto 0 auto;
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
