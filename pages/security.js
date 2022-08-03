import React from "react";
import styled from "@emotion/styled";
import Head from "next/head";

import Nav from "../shared/nav";
import Footer from "../shared/footer";

export async function getStaticProps() {
  return {
    props: {
      meta: {
        title: "Security",
        description: "Information on our platform security",
      },
    },
  };
}

const Security = () => {
  return (
    <div>
      <Nav />
      <Content>
        <h1>Security</h1>
        <h2>Organizational Security</h2>
        <h3>Information Security Program</h3>
        <p>
          We have an Information Security Program in place that is communicated
          throughout the organization. Our Information Security Program follows
          the criteria set forth by the SOC 2 Framework. SOC 2 is a widely known
          information security auditing procedure created by the American
          Institute of Certified Public Accountants.
        </p>

        <h3>Third-Party Audits</h3>
        <p>
          Our organization undergoes independent third-party assessments to test
          our security and compliance controls.
        </p>

        <h3>Third-Party Penetration Testing</h3>
        <p>
          We perform an independent third-party penetration at least annually to
          ensure that the security posture of our services is uncompromised.
        </p>

        <h3>Roles and Responsibilities</h3>
        <p>
          Roles and responsibilities related to our Information Security Program
          and the protection of our customer’s data are well defined and
          documented. Our team members are required to review and accept all of
          the security policies.
        </p>

        <h3>Security Awareness Training</h3>
        <p>
          Inngest employees are required to go through employee security
          awareness training covering industry standard practices and
          information security topics such as phishing and password management.
        </p>

        <h3>Confidentiality</h3>
        <p>
          All Inngest employees and contractors are required to sign and adhere
          to an industry standard confidentiality agreement prior to their first
          day of work
        </p>

        <h3>Background Checks</h3>
        <p>
          We perform background checks on all new employees in accordance with
          local laws.
        </p>

        <h2>Cloud Security</h2>

        <h3>Cloud Infrastructure Security</h3>
        <p>
          All of our services are hosted with Amazon Web Services (AWS) and
          Google Cloud Platform (GCP). They employ a robust security program
          with multiple certifications. For more information on our provider’s
          security processes, please visit{" "}
          <a
            href="http://aws.amazon.com/security/"
            rel="noopener noreferrer"
            target="_blank"
          >
            AWS Security
          </a>{" "}
          and{" "}
          <a
            href="https://cloud.google.com/security"
            rel="noopener noreferrer"
            target="_blank"
          >
            GCP Security
          </a>
          .
        </p>

        <h3>Data Hosting Security</h3>
        <p>
          All of our data is hosted on Amazon Web Services (AWS) databases.
          These databases are all located in the United States. Please reference
          the above vendor specific documentation linked above for more
          information.
        </p>

        <h3>Encryption at Rest</h3>
        <p>All databases are encrypted at rest.</p>

        <h3>Encryption in Transit</h3>
        <p>Our applications encrypt in transit with TLS/SSL only.</p>

        <h3>Vulnerability Scanning </h3>
        <p>
          We perform vulnerability scanning and actively monitor for threats.
        </p>

        <h3>Logging and Monitoring</h3>
        <p>We actively monitor and log various cloud services.</p>

        <h3>Business Continuity and Disaster Recovery</h3>
        <p>
          We use our data hosting provider’s backup services to reduce any risk
          of data loss in the event of a hardware failure. We utilize monitoring
          services to alert the team in the event of any failures affecting
          users.
        </p>

        <h3>Incident Response</h3>
        <p>
          We have a process for handling information security events which
          includes escalation procedures, rapid mitigation and communication.
        </p>

        <h2>Access Security</h2>

        <h3>Permissions and Authentication</h3>
        <p>
          Access to cloud infrastructure and other sensitive tools are limited
          to authorized employees who require it for their role.
        </p>
        <p>
          Where available we have Single Sign-on (SSO), 2-factor authentication
          (2FA) and strong password policies to ensure access to cloud services
          are protected.
        </p>

        <h3>Least Privilege Access Control</h3>
        <p>
          We follow the principle of least privilege with respect to identity
          and access management.
        </p>

        <h3>Quarterly Access Reviews</h3>
        <p>
          We perform quarterly access reviews of all team members with access to
          sensitive systems.
        </p>

        <h3>Password Requirements</h3>
        <p>
          All team members are required to adhere to a minimum set of password
          requirements and complexity for access.
        </p>

        <h3>Password Managers</h3>
        <p>
          All company issued laptops utilize a password manager for team members
          to manage passwords and maintain password complexity.
        </p>

        <h2>Vendor and Risk Management</h2>

        <h3>Annual Risk Assessments</h3>
        <p>
          We undergo at least annual risk assessments to identify any potential
          threats, including considerations for fraud.{" "}
        </p>

        <h3>Vendor Risk Management</h3>
        <p>
          Vendor risk is determined and the appropriate vendor reviews are
          performed prior to authorizing a new vendor.
        </p>

        <h2>Contact Us</h2>
        <p>
          If you have any questions, comments or concerns or if you wish to
          report a potential security issue, please contact
          security@inngest.com.
        </p>
      </Content>
      <Footer />
    </div>
  );
};

const Content = styled.div`
  max-width: 1200px;
  margin: 0 auto;

  padding: 0 20px;

  h1,
  h2,
  h3 {
    margin: 1em 0;
  }
`;

export default Security;
