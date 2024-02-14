import styled from "@emotion/styled";

export const PageTitle = styled.div`
  padding: 0 40px;
  display: flex;
  flex-direction: row;
  align-items: flex-start;
  justify-content: space-between;
  margin: 50px 0 40px;
  line-height: 1;

  > div:first-of-type {
    > span:first-of-type {
      text-transform: uppercase;
      opacity: 0.5;
      font-size: 11px;

      svg {
        position: relative;
        bottom: -3px;
        margin-right: 3px;
      }

      display: block;
      margin-bottom: 3px;
    }
  }

  h1 {
    line-height: 30px;

    small {
      font-size: 14px;
      font-weight: normal;
      margin: 0 0 0 10px;
      opacity: 0.5;
    }

    .tag {
      position: relative;
      top: -3px;
      margin-left: 15px;
    }
  }

  h1 + p {
    margin-top: 0.5rem;
    opacity: 0.6;
  }
`;

export const Aside = styled.div`
  display: flex;
  align-items: center;
  align-items: flex-start;
  margin: 0 0 3px 15px;

  h3 {
    opacity: 0.65;
  }

  > * + * {
    margin-left: 15px;
  }
`;

export default PageTitle;
