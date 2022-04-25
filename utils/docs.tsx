export type DocScope = {
  // If the slug contains a forward slahs (eg. foo/bar), this page will automatically
  // be nested under the page with a slug of "foo"
  slug: string;
  category: string;
  /** Featured image */
  image?: string;
  /** A meta description for the page itself */
  description?: string;
  /** Sub pages */
  pages?: DocScope[];
  // title is the title of the documentation page
  title: string;
  order: number;

  // reading is reading information automatically added when parsing content
  reading?: { text: string; time: number; words: number; minutes: number };
  // toc is the table of contents automatically added when parsing contnet
  toc?: Headings;
};

export type Headings = {
  [title: string]: {
    order: number;
    title: string;
    slug: string;
    subheadings: [{ title: string; slug: string }];
  };
};

type Doc = {
  slug: string;
  content: string;
  scope: DocScope;
};

export type Category = {
  title: string;
  order: number;
  pages: DocScope[];
};

export type Categories = { [title: string]: Category };

type Docs = {
  docs: { [slug: string]: Doc };
  slugs: string[];
  categories: Categories;
};

const CATEGORY_ORDER = {
  "What is Inngest?": 0,
  "Getting started": 1,
  "Sending & Managing Events": 2,
  "Managing workflows": 3,
};

export const getAllDocs = (() => {
  let memoizedDocs: Docs = {
    // docs maps docs by slug
    docs: {},
    slugs: [],
    categories: {},
  };

  return (): Docs => {
    if (
      memoizedDocs.slugs.length > 0 &&
      process.env.NODE_ENV === "production"
    ) {
      // memoizing in dev means you need to restart the server to see changes
      return memoizedDocs;
    }

    const fs = require("fs");
    const matter = require("gray-matter");
    const readingTime = require("reading-time");

    // parseDir is given the current directory path, then returns a function which
    // can read all files from the given basepath and processes the input.
    const parseDir = (basepath: string) => (fname: string) => {
      const fullpath = basepath + fname;

      if (fs.statSync(fullpath).isDirectory()) {
        // recurse into this directory with a new parse function using the extended
        // path.
        fs.readdirSync(fullpath).forEach(parseDir(fullpath + "/"));
        return;
      }

      const source = fs.readFileSync(fullpath);
      const { content, data: scope } = matter(source);

      if (scope.hide === true) {
        return;
      }

      memoizedDocs.slugs.push("/docs/" + scope.slug);
      memoizedDocs.docs[scope.slug] = {
        slug: scope.slug,
        content,
        scope: {
          ...scope,
          toc: getHeadings(content),
          reading: readingTime(content),
        },
      };
    };

    fs.readdirSync("./pages/docs/_docs/").forEach(
      parseDir("./pages/docs/_docs/")
    );

    const categories = {};

    // Iterate through each docs page and add the category.
    Object.values(memoizedDocs.docs).forEach((d: Doc) => {
      if (!d.scope.category) {
        console.warn("no category for doc", JSON.stringify(d.scope));
        return;
      }

      // Add category to list.
      if (!categories[d.scope.category]) {
        const order = CATEGORY_ORDER.hasOwnProperty(d.scope.category)
          ? CATEGORY_ORDER[d.scope.category]
          : 100;
        console.warn("no order for category", d.scope.category);
        categories[d.scope.category] = {
          title: d.scope.category,
          pages: [d.scope],
          order,
        };
      } else {
        categories[d.scope.category].pages.push(d.scope);
      }
    });

    memoizedDocs.categories = categories;

    return memoizedDocs;
  };
})();

export const getDocs = (slug: string): Doc | undefined => {
  const docs = getAllDocs();
  return docs.docs[slug];
};

const getHeadings = (content: string) => {
  // Get headers for table of contents.
  const headings = {};
  let h2 = null; // store the current heading we're in
  let order = 0;

  (content.match(/^###? (.*)/gm) || []).forEach((heading) => {
    const title = heading.replace(/^###? /, "");
    if (heading.indexOf("## ") === 0) {
      h2 = title;
      headings[title] = { title, slug: toSlug(title), subheadings: [], order };
      order++;
      return;
    }
    // add this subheading to the current heading list.
    (headings[h2]?.subheadings || []).push({ title, slug: toSlug(title) });
  });
  return headings;
};

const toSlug = (s: string) => {
  s = s.replace(/[^a-zA-Z0-9 :]/g, "");
  // rehype's `rehypeSlug` plugin converts "foo: one"  to "foo--one", and doesn't
  // remove multple slashes.  It does convert multiple spaces to just one slash.
  s = s.replace(/ +/g, "-");
  s = s.replace(/[:&]/g, "-");
  s = s.replace(/--/g, "-");
  return s.toLowerCase();
};
