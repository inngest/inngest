import styled from "@emotion/styled";

export default function UseCases() {
  return (
    <>
          <Title className="text-center">
            <h2>People use us to get stuff done</h2>
            <p>Say goodbye to fickle integrations and long development cycles.  Get stuff done in minutes using our platform.</p>
          </Title>
    <Grid>
      <Item>
        <div><img src="/assets/sync.png" /></div>
        <h3>Real-time sync</h3>
        <p>Enable real-time sync between any platform you integrate, with full support for anything custom.</p>
      </Item>

      <Item>
        <div><img src="/assets/churn.png" /></div>
        <h3>Churn management</h3>
        <p>Decrease churn and increase engagement by creating targeted workflows which run when users begin churning.</p>
      </Item>

      <Item>
        <div><img src="/assets/lead.png" /></div>
        <h3>Lead &amp; sales automation</h3>
        <p>Run custom workflows to rank, score, manage, and close your leads automatically.</p>
      </Item>

      <Item>
        <div><img src="/assets/ar.png" /></div>
        <h3>Dunning &amp; AR automation</h3>
        <p>Run custom logic when subscriptions or payments fail, capturing outstanding AR easily.</p>
      </Item>
    </Grid>
    </>
  );
}

const Title = styled.div`
  h2 { margin: 12rem 0 0; }
  h2 + p { opacity: .6; }
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

  > div:nth-child(1) { background: linear-gradient(-45deg, #51BEBA, #59A068); }
  > div:nth-child(2) { background: linear-gradient(-45deg, #9B76D9, #406E8B); }
  > div:nth-child(3) { background: linear-gradient(-45deg, #D6AB6D, #B94949); }
  > div:nth-child(4) { background: linear-gradient(-45deg, #D56C8F, #7A408B); }
`;

const Item = styled.div`
  > div {
    height: 120px;
    display: flex;
    align-items: center;
    margin: 0 0 2rem;
  }

  img { max-width: 100px; max-height: 90px; }

  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.15);
  padding: 2rem 2rem 4rem;
  font-weight: 500;
  background: #fff;
  border-radius: 5px;
`;
