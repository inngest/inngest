import styled from "@emotion/styled";
import Button from "./Button";

type Props = {
  small?: string;
  heading?: string;
  cta?: string;
  link?: string;
  ctaRef?: string;
  style?: any;
};

const Callout: React.FC<Props> = ({
  small,
  heading,
  cta,
  link,
  ctaRef,
  style,
}) => {
  return (
    <div className="grid" style={style}>
      <Content className="bg-primary">
        <div>
          <span>{small || "Now with zero yaml ;-)"}</span>
          <h2>{heading || "Deploy a serverless function in minutes."}</h2>
        </div>
        <Button
          kind="black"
          size="medium"
          href={`${link || "/sign-up"}${ctaRef ? "?ref=" + ctaRef : ""}`}
        >
          {cta || ">_ Start building"}
        </Button>
      </Content>
    </div>
  );
};

export default Callout;

const Content = styled.div`
  position: relative;
  grid-column: 3 / -2;
  grid-gap: 2rem;

  display: grid;
  grid-template-columns: 4fr 2fr;
  align-items: center;

  padding: var(--header-trailing-padding) 0;
  padding-right: var(--header-trailing-padding);

  border-top-right-radius: var(--border-radius);
  border-bottom-right-radius: var(--border-radius);

  color: var(--color-white);
  box-shadow: 0 0 4rem rgba(var(--black-rgb), 0.5);

  h2 {
    font-size: 1.3em;
  }

  span,
  button,
  a {
    font-family: var(--font-mono);
  }
  span {
    font-size: 16px;
  }

  button:hover,
  a:hover {
    background: var(--black);
    border-color: var(--black);
    box-shadow: 0 5px 25px rgba(var(--black-rgb), 0.6) !important;
  }

  &:before {
    display: block;
    content: "";
    background: var(--primary-color);
    left: -100%;
    position: absolute;
    height: 100%;
    width: 100%;
    top: 0;
  }

  @media (max-width: 800px) {
    grid-column: 2 / -2;
    grid-template-columns: 1fr;
    grid-gap: 1rem;
  }
`;
