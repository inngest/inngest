import Fuse from "fuse.js";
import { GetStaticProps } from "next";
import { useMemo, useState } from "react";
import Button from "src/shared/Button";
import Footer from "src/shared/footer";
import Nav from "src/shared/nav";
import { reqWithSchema } from "src/utils/fetch";
import { z } from "zod";

const miniCache: Record<string, any> = {};

interface Props {
  examples: Awaited<ReturnType<typeof getExamples>>;
}

export default function LibraryExamplesPage(props: Props) {
  const [search, setSearch] = useState("");
  const trimmedSearch = useMemo(() => search.trim(), [search]);

  const fuse = useMemo(() => {
    return new Fuse(props.examples, {
      keys: ["name", "description"],
      findAllMatches: true,
    });
  }, [props.examples]);

  const examples = useMemo(() => {
    if (!trimmedSearch) {
      return props.examples;
    }

    return fuse.search(trimmedSearch).map(({ item }) => item);
  }, [fuse, trimmedSearch]);

  return (
    <div>
      <Nav sticky nodemo />
      <div className="container mx-auto pt-32 pb-24 flex flex-row">
        <div className="text-center px-6 max-w-4xl mx-auto flex flex-col space-y-6">
          <h1>Examples</h1>
          <p className="subheading">...</p>
        </div>
      </div>

      <div className="container mx-auto max-w-lg mb-6 px-12">
        <input
          type="text"
          placeholder="Search..."
          className="w-full bg-gray-100 border border-gray-200"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      <div className="background-grid-texture">
        <div className="container mx-auto p-12 grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-12">
          {examples.map((example) => (
            <div
              key={example.id}
              className="rounded-lg border border-gray-200 p-6 flex flex-col space-y-2 bg-white"
            >
              <div className="font-semibold">{example.name}</div>
              <div>{example.description}</div>
              <Button
                kind="outline"
                size="medium"
                className="inline-block"
                href={`/examples/${example.id}?ref=examples`}
              >
                Explore
              </Button>
            </div>
          ))}
        </div>
      </div>
      <Footer />
    </div>
  );
}

export const getStaticProps: GetStaticProps = async () => {
  return { props: { examples: await getExamples() } };
};

export const getExamples = async () => {
  const latestCommit = await reqWithSchema(
    "https://api.github.com/repos/inngest/inngest/commits/main",
    z.object({
      commit: z.object({ tree: z.object({ url: z.string() }) }),
    }),
    miniCache
  );

  const commitContents = await reqWithSchema(
    latestCommit.commit.tree.url,
    z.object({
      tree: z.array(treeSchema),
    }),
    miniCache
  );

  const examplesPath = commitContents.tree.find(
    ({ path, type }) => path === "examples" && type === "tree"
  );

  if (!examplesPath?.url) {
    throw new Error("Could not find examples path");
  }

  const examplesTree = await reqWithSchema(
    examplesPath.url,
    z.object({
      tree: z.array(treeSchema),
    }),
    miniCache
  );

  const exampleNodes = examplesTree.tree.filter(({ type }) => type === "tree");

  const examples = await Promise.all(
    exampleNodes.map(async (node) => {
      const example = await reqWithSchema(
        node.url,
        z.object({ tree: z.array(treeSchema) }),
        miniCache
      );

      const inngestJsonNode = example.tree.find(
        ({ path, type }) => path === "inngest.json" && type === "blob"
      );

      if (!inngestJsonNode?.url) {
        throw new Error("Could not find inngest.json in example");
      }

      const inngestJsonRaw = await reqWithSchema(
        inngestJsonNode.url,
        blobSchema,
        miniCache
      );

      return z
        .object({
          id: z.string(),
          tree: z.array(treeSchema),
          name: z.string(),
          description: z.string().optional(),
          tags: z.array(z.string()).optional(),
        })
        .parse({
          ...JSON.parse(
            inngestJsonRaw.encoding === "base64"
              ? Buffer.from(inngestJsonRaw.content, "base64").toString()
              : inngestJsonRaw.content
          ),
          id: node.path,
          tree: example.tree,
        });
    })
  );

  return examples;
};

const treeSchema = z.object({
  path: z.string(),
  sha: z.string().or(z.null()),
  type: z.union([z.literal("blob"), z.literal("tree"), z.literal("commit")]),
  url: z.string(),
});

export const blobSchema = z.object({
  content: z.string(),
  encoding: z.union([z.literal("utf-8"), z.literal("base64")]),
});
