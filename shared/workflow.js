import styled from "@emotion/styled";
import Action, { Trigger, Empty, If } from "./action";

export default function Workflow({ style, className }) {
  return (
    <Wrapper style={style} className={className}>
      <BG />
      <Flow>
        <Row>
          <Trigger name="stripe.invoice.paid" className="conn-bottom" />
        </Row>
        <Rule />
        <Row>
          <Action name="Update account in Salesforce" subtitle="Set the account's value"
                    icon="/icons/sf-cloud.svg" className="conn-top conn-bottom" />
          <Action name="Send a slack notification" subtitle="In #new-payments" className="conn-top conn-bottom"/>
          <Action name="Send a receipt" subtitle="Via Mailchimp" className="conn-top conn-bottom" />
        </Row>
        <Row className="expression">
          <If expression="Amount >= $500" className="conn-bottom conn-top" />
          <Action name="Update TAM dashboard" subtitle="Within Trello" className="conn-top" />
          <If expression="If email bounces within 1 day" className="conn-bottom conn-top" />
        </Row>
        <Row>
          <Action name="Add to VIP list" subtitle="Via Mailchimp" className="conn-top" />
          <Empty />
          <Action name="Create support issue" subtitle="Within Zendesk" className="conn-top" />
        </Row>
      </Flow>
    </Wrapper>
  );
}


const Wrapper = styled.div`
  position: relative;
  height: 400px;
  z-index: 1;

  /*
  transform: perspective(1500px) rotateX(51deg) rotateZ(43deg);
  transform-style: preserve-3d;
  */
  transform: perspective(2500px) rotateX(10deg) rotateY(346deg) rotateZ(4deg);
  transform: perspective(2500px) rotateX(31deg) rotateY(346deg) rotateZ(18deg);
`;

const Row = styled.div`
  display: flex;
  justify-content: center;
  margin: 0 0 40px;

  &.expression {
  }

  > div + div {
    margin-left: 40px;
  }
`

const lineCSS = `
  background: #ddd;
  opacity: 0.3;
`

const BG = styled.div`
  position: absolute;
  background: #fff;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.1);
  height: 420px;
  width: 920px;
  border-radius: 20px;
  opacity: 0.2;
  filter: blur(120px);
`;

const Flow = styled.div`
  position: absolute;

  top: -40px;
  left: -40px;

  /*
  top: -20px;
  left: -20px;
  */

  .action {
    box-shadow: 8px 10px 30px rgb(0 0 0 / 25%),
      1px 1px 0px 0px #fff,
      2px 2px 0px 0px #fff,
      3px 5px 0px 0px #fff;
  }

  .conn-top:before {
    width: 1px;
    content: "";
    display: block;
    position: absolute;
    left: 50%;
    top: -21px;
    height: 20px;
    ${lineCSS};
  }

  .conn-bottom:after {
      width: 1px;
      content: "";
      display: block;
      position: absolute;
      left: 50%;
      bottom: -21px;
      height: 20px;
      ${lineCSS};
  }

  .if {
    box-shadow: 8px 10px 30px rgba(0, 0, 0, 0.05);
    height: 60px;
    margin-top: 10px;

    &.conn-top:before {
      height: 30px;
      top: -31px;
    }
    &.conn-bottom:after {
      height: 30px;
      bottom: -31px;
    }
  }
`;

const Rule = styled.div`
  ${lineCSS};
  height: 1px;
  width: 641px;
  margin: -20px 0 20px 140px;
`;
