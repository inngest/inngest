import styled from "@emotion/styled";

export const IntegrationType = {
  EVENTS: "Events",
  FUNCTIONS: "Functions", // SDK?
};

export type IntegrationType =
  typeof IntegrationType[keyof typeof IntegrationType];

type Props = {
  name: string;
  category: string;
  // Logo represents the src of the logo.
  logo: string;
  type?: IntegrationType[];
};

export const Integration: React.FC<Props> = (props) => {
  return (
    <Wrapper>
      <img src={props.logo} alt={props.name} />
      <div>
        <p className="name">{props.name}</p>
        <span>{props.category}</span>
        {props.type.map((typ) => (
          <span className="type" key={typ}>
            {typ}
          </span>
        ))}
      </div>
    </Wrapper>
  );
};

export default Integration;

const Wrapper = styled.div`
  background: var(--black);
  border-radius: var(--border-radius);
  display: grid;
  align-items: center;
  grid-template-columns: minmax(2rem, 8rem) minmax(10rem, 1fr);

  /* This automatically has 1rem added to it, as the image is pulled left 1rem */
  grid-gap: 0.5rem;

  margin-left: 1rem;

  img {
    margin-left: -1rem;
    width: 100%;
  }

  div {
    padding: 2rem 2rem 2rem 0;
  }

  .name {
    font-weight: bold;
    font-size: 1.5rem;
  }

  span {
    display: block;
    margin: 0.25rem 0;
  }

  .type {
    background: var(--gray);
    padding: 0.5rem;
    display: inline-block;
    font-family: var(--font-mono);
    border-radius: var(--border-radius);
    font-size: 14px;
  }

  @media (max-width: 800px) {
    grid-template-columns: 3.5rem minmax(10rem, 1fr);

    .name { font-size: 1.2rem; }
    span { font-size: 1rem; line-height: 1.2; }
  }
`;
