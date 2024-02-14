import React from "react";
import styles from "../styles/Press.module.css";

const ASSETS = [
  {
    name: "Logo (svg)",
    src: "/logo.svg",
  },
  {
    name: "Logo (png)",
    src: "/logo.png",
  },
  {
    name: "Icon - dark (png)",
    src: "/icon-dark.png",
  },
  {
    name: "Icon - light (png)",
    src: "/icon-light.png",
  },
  {
    name: "Icon - transparent (png)",
    src: "/icon-light-transparent.png",
  },
];

const Press = () => {
  return (
    <div className={styles.container}>
      <h1>Inngest Press Kit</h1>
      <h4>When referencing us on the web, feel free to use these assets.</h4>
      <div className={styles.assets}>
        {ASSETS.map((asset) => (
          <div className={styles.asset}>
            <a href={asset.src} download>
              <img src={asset.src} />
            </a>
            <div>{asset.name}</div>
          </div>
        ))}
      </div>
    </div>
  );
};
export default Press;
