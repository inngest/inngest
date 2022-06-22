import React from "react";
import styled from "@emotion/styled";

const ContentBlock: React.FC<{
  layout?: "default" | "reverse";
  heading: string | React.ReactNode;
  text: string | React.ReactNode;
  icon?: React.ReactNode;
  image?: string;
  imageSize?: "default" | "full";
}> = ({
  layout = "default",
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
      {layout === "reverse" && imageBox}
      <div className="content">
        {icon ? icon : ""}
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
  margin: 6rem auto;

  .content {
    grid-column: ${({ layout }) => (layout === "default" ? "2/6" : "6/10")};
    padding: 2rem 0;
  }

  svg {
    margin-bottom: 1rem;
  }

  h3 {
    margin-bottom: 1rem;
    font-size: 1.6rem;
  }
  p {
    font-size: 0.9rem;
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
`;

export default ContentBlock;
