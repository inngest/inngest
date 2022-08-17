import fetch from "cross-fetch";
import { GetStaticPaths, GetStaticProps } from "next";
import { z } from "zod";

const miniCache: Record<string, any> = {};

const treeSchema = z.object({
  path: z.string(),
  sha: z.string().or(z.null()),
  type: z.union([z.literal("blob"), z.literal("tree"), z.literal("commit")]),
  url: z.string(),
});

const blobSchema = z.object({
  content: z.string(),
  encoding: z.union([z.literal("utf-8"), z.literal("base64")]),
});

interface Props {
  name: string;
  description?: string;
  readme?: string;
}

export default function LibraryExamplePage(props: Props) {
  return <div>{JSON.stringify(props, null, 2)}</div>;
}

export const getStaticPaths: GetStaticPaths = async () => {
  const examplesContents = await getExamplesContents();

  const paths = examplesContents.tree.reduce((acc, { path, sha }) => {
    if (!sha) {
      return acc;
    }

    return [...acc, { params: { example: path } }];
  }, []);

  return { paths, fallback: false };
};

export const getStaticProps: GetStaticProps = async (ctx) => {
  const examplesContents = await getExamplesContents();

  const target = examplesContents.tree.find(
    ({ path }) => path === ctx.params.example
  );

  if (!target.sha) {
    throw new Error("Could not find example");
  }

  const example = await reqWithSchema(
    `https://api.github.com/repos/inngest/inngest/git/trees/${target.sha}`,
    z.object({
      tree: z.array(treeSchema),
    })
  );

  const inngestJsonNode = example.tree.find(
    ({ path, type }) => path === "inngest.json" && type === "blob"
  );

  if (!inngestJsonNode.url) {
    throw new Error("Could not find inngest.json in example");
  }

  const inngestJsonRaw = await reqWithSchema(inngestJsonNode.url, blobSchema);

  const inngestJson = z
    .object({
      name: z.string(),
      description: z.string().optional(),
    })
    .parse(
      JSON.parse(
        inngestJsonRaw.encoding === "base64"
          ? Buffer.from(inngestJsonRaw.content, "base64").toString()
          : inngestJsonRaw.content
      )
    );

  let readme: string | undefined;
  const readmeNode = example.tree.find(
    ({ path, type }) => path === "README.md" && type === "blob"
  );

  if (readmeNode.url) {
    const readmeRaw = await reqWithSchema(readmeNode.url, blobSchema);

    readme =
      readmeRaw.encoding === "base64"
        ? Buffer.from(readmeRaw.content, "base64").toString()
        : readmeRaw.content;
  }

  return {
    props: {
      ...inngestJson,
      readme,
      meta: {
        title: `Example: ${inngestJson.name}`,
        description: inngestJson.description || "Some default meta description",
      },
    },
  };
};

const reqWithSchema = async <T extends z.ZodTypeAny>(
  url: string,
  schema: T
): Promise<z.output<T>> => {
  const json =
    miniCache[url] ||
    (await fetch(url, {
      headers: {
        "User-Agent": "inngest",
      },
    })
      .then((res) => res.json())
      .then((data) => {
        miniCache[url] = data;
        return data;
      }));

  return schema.parse(json);
};

const getExamplesContents = async () => {
  const latestCommit = await reqWithSchema(
    "https://api.github.com/repos/inngest/inngest/commits/main",
    z.object({
      commit: z.object({ tree: z.object({ url: z.string() }) }),
    })
  );

  const commitContents = await reqWithSchema(
    latestCommit.commit.tree.url,
    z.object({
      tree: z.array(treeSchema),
    })
  );

  const examplesPath = commitContents.tree.find(
    ({ path, type }) => path === "examples" && type === "tree"
  );

  if (!examplesPath.url) {
    throw new Error("Could not find examples path");
  }

  return reqWithSchema(
    examplesPath.url,
    z.object({
      tree: z.array(treeSchema),
    })
  );
};
