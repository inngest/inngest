import type { NextApiRequest, NextApiResponse } from "next";
import RSS from "rss";

import { loadMarkdownFilesMetadata } from "../../utils/markdown";
import { type BlogPost } from "../blog";

export default async (req: NextApiRequest, res: NextApiResponse<string>) => {
  const posts = await loadMarkdownFilesMetadata<BlogPost>("blog/_posts");

  const feed = new RSS({
    title: "Inngest Product & Engineering Blog",
    description:
      "Updates from the Inngest team about our product, engineering, and community",
    feed_url: `${process.env.NEXT_PUBLIC_HOST}/rss.xml`,
    site_url: process.env.NEXT_PUBLIC_HOST,
    image_url: `${process.env.NEXT_PUBLIC_HOST}/${process.env.NEXT_PUBLIC_FAVICON}`,
    language: "en-us",
  });

  posts
    .filter((post) => !post.hide)
    .forEach((post) => {
      feed.item({
        title: post.heading,
        description: post.subtitle,
        author: post.author,
        date: post.date,
        url: `${process.env.NEXT_PUBLIC_HOST}/blog/${post.slug}`,
        categories: post.tags || [],
      });
    });

  const xml = feed.xml();

  res.setHeader("Content-Type", "text/xml");
  res.setHeader("Cache-Control", "s-maxage=360, stale-while-revalidate");
  res.write(xml);
  res.end();
};
