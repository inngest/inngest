import styled from "@emotion/styled";

const Content = styled.div`
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 18px;

  @media only screen and (max-width: 800px) {
    padding: 0 20px;
  }

  > header {
    margin: 16vh auto 12vh;
    max-width: 70vw;
  }
`;

export default Content;
