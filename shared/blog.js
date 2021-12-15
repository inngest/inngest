import styled from "@emotion/styled";

export const Wrapper = styled.div`
  background: var(--bg-gradient);
`;

export const Inner = styled.div`
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  padding: 120px 0;

  p,
  h1,
  h2,
  ul,
  div {
    grid-column: 2/-2;
  }

  > p {
    margin-bottom: 0.25rem;
  }
  > ul {
    margin: 2rem 0 0;
  }

  p,
  ul,
  li {
    font-size: 18px;
    line-height: 1.7;
  }

  h1 {
    margin: 0 0 2rem;
  }

  h2 {
    margin: 3.5rem 0 1rem;
    font-weight: 600;
    font-size: 1.65rem;
  }

  .blog--date {
    font-size: 14px;
    opacity: 0.6;
    margin: 1rem 0 3rem;
    padding: 0 0 0 1rem;
    border-left: 2px solid var(--light-grey);
  }

  .blog--callout {
    font-weight: 500;

    box-sizing: content-box;
    padding: 2rem;
    margin: 8vh 0;
    border-radius: 10px;

    background-image: linear-gradient(
      -45deg,
      rgba(255, 255, 255, 0.1) 25%,
      transparent 25%,
      transparent 50%,
      rgba(255, 255, 255, 0.1) 50%,
      rgba(255, 255, 255, 0.1) 75%,
      transparent 75%,
      transparent
    );
    background-size: 5px 5px;
  }
`;
