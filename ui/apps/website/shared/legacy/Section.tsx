import styled from "@emotion/styled";

const Section = styled.section<{ theme?: "dark" | "light" }>`
  margin: 0 auto;
  padding: 5rem 0;
  background-color: ${({ theme }) =>
    theme === "dark" ? "var(--black)" : "inherit"};
  color: ${({ theme }) =>
    theme === "dark" ? "var(--color-white)" : "inherit"};

  header {
    text-align: center;
  }

  h2 {
    font-size: 2.5rem;

    svg {
      display: inline-block;
      margin-right: 0.1rem;
      vertical-align: top;
      position: relative;
      top: 0.2rem;
    }
  }

  .subheading {
    margin: 1em auto;
    max-width: 900px;
    font-size: 1rem;
    line-height: 1.6em;
  }

  .cta-container {
    text-align: center;

    .button {
      display: inline-flex;
    }
  }

  // A 3-up content grid
  .content-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    grid-gap: 2rem;
    margin: 5rem 0;

    h3 {
      margin-bottom: 1rem;
      font-style: italic;
    }
  }

  @media (max-width: 960px) {
    .content-grid {
      margin: 3rem 0;
      grid-template-columns: repeat(5, 1fr);

      > div:nth-of-type(1) {
        grid-column: 1/4;
      }
      > div:nth-of-type(2) {
        grid-column: 2/5;
      }
      > div:nth-of-type(3) {
        grid-column: 3/6;
      }
    }
  }

  @media (max-width: 800px) {
    padding: 4rem 0;
    header {
      padding: 0 2rem;
    }
    h2 {
      font-size: 2rem;

      svg {
        display: block;
        margin: 0 auto;
      }
    }

    .content-grid {
      display: flex;
      padding: 0 1rem;
      flex-direction: column;
    }
  }
`;

export default Section;
