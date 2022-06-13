// Render all the social previews to static images during build
import fs from "fs";
import { getAllDocs } from "../utils/docs";
import renderReactToPng from "../utils/renderReactToPng";
import SocialPreview from "../shared/Docs/SocialPreview";

const OUTPUT_DIR = "./out/assets/social-previews";

const { docs } = getAllDocs();

async function renderAll() {
  fs.mkdirSync(OUTPUT_DIR);

  for await (let [slug, scope] of Object.entries(docs)) {
    const flattenedSlug = slug.replace(/\//, "--");
    const filename = `${OUTPUT_DIR}/${flattenedSlug}.png`;
    const titleWithCategory =
      scope.scope.category !== scope.scope.title
        ? `${scope.scope.category}: ${scope.scope.title}`
        : scope.scope.title;
    const image = await renderReactToPng({
      Component: SocialPreview,
      props: { title: titleWithCategory },
    });
    fs.writeFileSync(filename, image);
    console.log(`Rendered: ${filename}`);
  }
}

renderAll();
