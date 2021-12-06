import styled from "@emotion/styled";

const Content = styled.div`
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 18px;
  position: relative;

  @media only screen and (max-width: 800px) {
    padding: 0 20px;
  }

  > header {
    display: block;
    padding: 16vh 0 12vh;
    margin: 0 auto;
    max-width: min(70vw, 1000px);
  }


  &.top-gradient::before {
    display: block;
    width: 100%;
    min-height: 15vh;
    content: "";
    position: absolute;
    top: 12vh;
    opacity: 0.3;
    background: radial-gradient(52.28% 118.04% at 50% 1.76%, #1B4074 0%, rgba(34, 40, 102, 0) 100%);
    filter: drop-shadow(0px -30px 80px rgba(0, 0, 0, 0.25));
    border-radius: 50px;
  }
`;

export default Content;
