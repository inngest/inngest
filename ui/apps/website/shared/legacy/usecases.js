import styled from "@emotion/styled";

export default function UseCases() {
  return (
    <>
      <Title className="text-center">
        <h2>People use us to get stuff done</h2>
        <p>
          Say goodbye to fickle integrations and long development cycles. Get
          stuff done in minutes using our platform.
          <br />
          Here's a few things that people use us for:
        </p>
      </Title>
      <Grid>
        <Item>
          <div>
            <img src="/assets/sync.png" />
          </div>
          <h3>Real-time sync</h3>
          <p>
            Enable real-time sync between any platform you integrate, with full
            support for anything custom.
          </p>
        </Item>

        <Item>
          <div>
            <img src="/assets/churn.png" />
          </div>
          <h3>User flows</h3>
          <p>
            Create targeted workflows which run when users do things or{" "}
            <i>don't</i> - by monitoring for the <b>absence of events</b>.
          </p>
        </Item>

        <Item>
          <div>
            <img src="/assets/lead.png" />
          </div>
          <h3>Lead &amp; sales automation</h3>
          <p>
            Run custom workflows to rank, score, manage, and close your leads
            automatically.
          </p>
        </Item>

        <Item>
          <div>
            <img src="/assets/ar.png" />
          </div>
          <h3>Billing &amp; AR automation</h3>
          <p>
            Run custom logic when subscriptions charge - or payments fail,
            capturing outstanding AR easily.
          </p>
        </Item>
      </Grid>
    </>
  );
}

const Title = styled.div`
  h2 {
    margin: 12rem 0 0;
  }
  h2 + p {
    opacity: 0.6;
  }
`;

const Grid = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-gap: 3rem;
  padding: 0 12rem;
  margin: 5rem 0;
  color: #fff;

  h3 {
    font-weight: 800;
  }

  > div:nth-child(1) {
    background: linear-gradient(-45deg, #51beba, #59a068);
  }
  > div:nth-child(2) {
    background: linear-gradient(-45deg, #9b76d9, #406e8b);
  }
  > div:nth-child(3) {
    background: linear-gradient(-45deg, #d6ab6d, #b94949);
  }
  > div:nth-child(4) {
    background: linear-gradient(-45deg, #d56c8f, #7a408b);
  }
`;

const Item = styled.div`
  > div {
    height: 120px;
    display: flex;
    align-items: center;
    margin: 0 0 2rem;
  }

  img {
    max-width: 100px;
    max-height: 90px;
  }

  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.15);
  padding: 2rem 2rem 4rem;
  font-weight: 500;
  background: #fff;
  border-radius: 5px;
`;
