import { mdxAnnotations } from "mdx-annotations";
import remarkGfm from "remark-gfm";
import remarkCodeTitles from "remark-code-titles";

export const remarkPlugins = [
  mdxAnnotations.remark,
  remarkGfm,
  remarkCodeTitles,
];
1;
