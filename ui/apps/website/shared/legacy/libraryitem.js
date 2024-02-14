import styled from "@emotion/styled";

export const Inner = styled.div`
  box-sizing: border-box;
  padding: 65px 0;
  min-height: calc(100vh - 270px);

  .back {
    font-size: 14px;
    display: block;
    margin: 0 0 1rem;
  }

  h2 {
    margin: 0;
  }

  h2 + p {
    margin: 0.35rem 0 0.5rem;
    opacity: 0.6;
  }
`;

export const WorkflowContent = styled.div`
  display: grid;
  color: var(--dark-grey);

  /*
  grid-template-columns: auto 180px;
  gap: 40px;
  */

  margin: 80px 0;

  .editor {
    min-height: 550px;
  }
`;

export const Description = styled.div`
  h1,
  h2,
  h3,
  h4,
  h5,
  h6 {
    font-weight: 600;
  }
  h1 {
    font-size: 1.35rem;
  }
  h2,
  h3,
  h4 {
    font-size: 1rem;
  }

  h1 {
    margin: 2rem 0 0.5rem;
  }
  h2 {
    margin: 3rem 0 0.25rem;
  }
`;
