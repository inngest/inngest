import React from "react";
import styled from "@emotion/styled";

const ContentBlock: React.FC<{
  layout?: "default" | "reverse";
  preline?: string | React.ReactNode;
  heading: string | React.ReactNode;
  text: string | React.ReactNode;
  icon?: React.ReactNode;
  image?: string;
  imageSize?: "default" | "full";
}> = ({
  layout = "default",
  preline,
  heading,
  text,
  icon,
  image,
  imageSize = "default",
}) => {
  const imageBox = (
    <div className="image" style={{ backgroundImage: `url(${image})` }}></div>
  );
  return (
    <Block layout={layout} imageSize={imageSize}>
      <img src={image} className="image-mobile" />
      {layout === "reverse" && imageBox}
      <div className="content">
        {icon ? icon : ""}
        {preline ? <div className="preline">{preline}</div> : ""}
        <h3>{heading}</h3>
        <p>{text}</p>
      </div>
      {layout === "default" && imageBox}
    </Block>
  );
};

const Block = styled.div<{
  layout: "default" | "reverse";
  imageSize: "default" | "full";
}>`
  max-width: 1200px;
  display: grid;
  grid-template-columns: repeat(
    11,
    1fr
  ); // 11 so nothing is perfectly down the middle of the page
  margin: 4rem auto;

  .content {
    grid-column: ${({ layout }) => (layout === "default" ? "2/6" : "6/10")};
    padding: 2rem 0;
  }

  .image-mobile {
    display: none;
  }

  svg {
    margin-bottom: 1rem;
  }

  .preline {
    font-size: 0.8rem;
    margin-bottom: 1rem;
  }

  h3 {
    margin-bottom: 1rem;
    font-size: 1.6rem;
  }
  p {
    font-size: 0.8rem;
  }

  .image {
    grid-column: ${({ layout, imageSize }) => {
      const cols = {
        // layout imageSize
        "default default": "7/10",
        "default full": "7/12",
        "reverse default": "2/5",
        "reverse full": "1/5",
      };
      return cols[`${layout} ${imageSize}`];
    }};
    background-size: contain;
    background-repeat: no-repeat;
    background-position: ${({ layout }) =>
      layout === "default" ? "left center" : "right center"};
  }

  @media (max-width: 800px) {
    margin: 3rem 10%;
    grid-template-columns: 1fr;
    grid-auto-rows: auto;

    .content {
      grid-column: 1;
      grid-row: 2;
      padding-bottom: 0;
    }
    // Sizing is easier with an image tag in stacked layouts
    .image-mobile {
      display: block;
    }
    .image {
      display: none;
    }
  }
`;

export default ContentBlock;
