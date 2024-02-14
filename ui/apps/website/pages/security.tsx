import React from "react";
import Header from "src/shared/Header";
import Container from "src/shared/layout/Container";
import Footer from "src/shared/Footer";
import { Button } from "src/shared/Button";

export async function getStaticProps() {
  return {
    props: {
      meta: {
        title: "Security",
        description: "Information on our platform security",
      },
      designVersion: "2",
    },
  };
}

const Security = () => {
  return (
    <div className="font-sans">
      <Header />
      <Container>
        <article>
          <main className="m-auto max-w-[80ch] pt-16">
            <header className="pt-12 lg:pt-24 m-auto">
              <h1 className="text-white font-medium text-2xl md:text-4xl xl:text-5xl mb-2 md:mb-4 tracking-tighter lg:leading-loose">
                Security
              </h1>
              <Button href="#contact-us" arrow="right">
                Report a security issue
              </Button>
            </header>
            <div className="my-20 mx-auto prose prose-img:rounded-lg prose-code:bg-slate-800 prose-code:tracking-tight text-slate-300 prose-a:text-indigo-400 prose-a:no-underline hover:prose-a:underline hover:prose-a:text-white prose-a:font-medium prose-a:transition-all prose-invert">
              <h2>Organizational Security</h2>
              <h3>Information Security Program</h3>
              <p>
                We have an Information Security Program in place that is
                communicated throughout the organization. Our Information
                Security Program follows the criteria set forth by the SOC 2
                Framework. SOC 2 is a widely known information security auditing
                procedure created by the American Institute of Certified Public
                Accountants.
              </p>

              <h3>Third-Party Audits</h3>
              <p>
                Our organization undergoes independent third-party assessments
                to test our security and compliance controls.
              </p>

              <h3>Third-Party Penetration Testing</h3>
              <p>
                We perform an independent third-party penetration at least
                annually to ensure that the security posture of our services is
                uncompromised.
              </p>

              <h3>Roles and Responsibilities</h3>
              <p>
                Roles and responsibilities related to our Information Security
                Program and the protection of our customer’s data are well
                defined and documented. Our team members are required to review
                and accept all of the security policies.
              </p>

              <h3>Security Awareness Training</h3>
              <p>
                Inngest employees are required to go through employee security
                awareness training covering industry standard practices and
                information security topics such as phishing and password
                management.
              </p>

              <h3>Confidentiality</h3>
              <p>
                All Inngest employees and contractors are required to sign and
                adhere to an industry standard confidentiality agreement prior
                to their first day of work
              </p>

              <h3>Background Checks</h3>
              <p>
                We perform background checks on all new employees in accordance
                with local laws.
              </p>

              <h2>Cloud Security</h2>

              <h3>Cloud Infrastructure Security</h3>
              <p>
                All of our services are hosted with Amazon Web Services (AWS)
                and Google Cloud Platform (GCP). They employ a robust security
                program with multiple certifications. For more information on
                our provider’s security processes, please visit{" "}
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
                All of our data is hosted on Amazon Web Services (AWS)
                databases. These databases are all located in the United States.
                Please reference the above vendor specific documentation linked
                above for more information.
              </p>

              <h3>Encryption at Rest</h3>
              <p>All databases are encrypted at rest.</p>

              <h3>Encryption in Transit</h3>
              <p>Our applications encrypt in transit with TLS/SSL only.</p>

              <h3>Vulnerability Scanning </h3>
              <p>
                We perform vulnerability scanning and actively monitor for
                threats.
              </p>

              <h3>Logging and Monitoring</h3>
              <p>We actively monitor and log various cloud services.</p>

              <h3>Business Continuity and Disaster Recovery</h3>
              <p>
                We use our data hosting provider’s backup services to reduce any
                risk of data loss in the event of a hardware failure. We utilize
                monitoring services to alert the team in the event of any
                failures affecting users.
              </p>

              <h3>Incident Response</h3>
              <p>
                We have a process for handling information security events which
                includes escalation procedures, rapid mitigation and
                communication.
              </p>

              <h2>Access Security</h2>

              <h3>Permissions and Authentication</h3>
              <p>
                Access to cloud infrastructure and other sensitive tools are
                limited to authorized employees who require it for their role.
              </p>
              <p>
                Where available we have Single Sign-on (SSO), 2-factor
                authentication (2FA) and strong password policies to ensure
                access to cloud services are protected.
              </p>

              <h3>Least Privilege Access Control</h3>
              <p>
                We follow the principle of least privilege with respect to
                identity and access management.
              </p>

              <h3>Quarterly Access Reviews</h3>
              <p>
                We perform quarterly access reviews of all team members with
                access to sensitive systems.
              </p>

              <h3>Password Requirements</h3>
              <p>
                All team members are required to adhere to a minimum set of
                password requirements and complexity for access.
              </p>

              <h3>Password Managers</h3>
              <p>
                All company issued laptops utilize a password manager for team
                members to manage passwords and maintain password complexity.
              </p>

              <h2>Vendor and Risk Management</h2>

              <h3>Annual Risk Assessments</h3>
              <p>
                We undergo at least annual risk assessments to identify any
                potential threats, including considerations for fraud.{" "}
              </p>

              <h3>Vendor Risk Management</h3>
              <p>
                Vendor risk is determined and the appropriate vendor reviews are
                performed prior to authorizing a new vendor.
              </p>

              <h2 id="contact-us" className="scroll-mt-32">
                Contact Us
              </h2>
              <p>
                If you have any questions, comments or concerns or if you wish
                to report a potential security issue, please contact{" "}
                <a href="mailto:security@inngest.com">security@inngest.com</a>.
              </p>
              <p>
                In order to ensure security reports are actionable and to
                prevent our security inbox from being inundated with invalid
                reports, please review{" "}
                <a href="https://bughunters.google.com/learn/invalid-reports/5374985771941888">
                  the list of non-qualifying reports on Google's website
                </a>
                . If your report falls into one of their categories, i.e.{" "}
                <em>"CSRF that requires the knowledge of a secret"</em>, there
                is no need to report it.
              </p>
            </div>
          </main>
        </article>
      </Container>
      <Footer />
    </div>
  );
};

export default Security;
