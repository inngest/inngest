
export type DocScope = {
  // If the slug contains a forward slahs (eg. foo/bar), this page will automatically
  // be nested under the page with a slug of "foo"
  slug: string;
  category: string;
  // title is the title of the documentation page
  title: string;
  order: number;

  // reading is reading information automatically added when parsing content
  reading?: { text: string, time: number, words: number, minutes: number };
  // toc is the table of contents automatically added when parsing contnet
  toc?: Headings;
}

type Headings = {
  [title: string]: {
    order: number;
    title: string;
    slug: string;
    subheadings: [{ title: string; slug: string; }]
  }
};

type Doc = {
  slug: string,
  content: string,
  scope: DocScope,
}

export type Category = {
  title: string;
  pages: DocScope[];
}

export type Categories = { [title: string]: Category }

type Docs = {
  docs: { [slug: string]: Doc }
  slugs: string[];
  categories: Categories;
}

export const getAllDocs = (() => {
  let memoizedDocs: Docs = {
    // docs maps docs by slug 
    docs: {},
    slugs: [],
    categories: {},
  }

  return (): Docs => {
    if (memoizedDocs.slugs.length > 0 && process.env.NODE_ENV === "production") {
    // memoizing in dev means you need to restart the server to see changes
      return memoizedDocs;
    }
    
    const fs = require('fs');
    const matter = require('gray-matter');
    const readingTime = require('reading-time');

    // parseDir is given the current directory path, then returns a function which
    // can read all files from the given basepath and processes the input.
    const parseDir = (basepath: string) => (fname: string) => {
      const fullpath = basepath + fname;

      if (fs.statSync(fullpath).isDirectory()) {
        // recurse into this directory with a new parse function using the extended
        // path.
        fs.readdirSync(fullpath).forEach(parseDir(fullpath + "/"));
        return
      }

      const source = fs.readFileSync(fullpath);
      const { content, data: scope } = matter(source)

      memoizedDocs.slugs.push("/docs/" + scope.slug);
      memoizedDocs.docs[scope.slug] = {
        slug: scope.slug,
        content,
        scope: {
          ...scope,
          toc: getHeadings(content),
          reading: readingTime(content),
        },
      }
    }

    fs.readdirSync("./pages/docs/_docs/").forEach(parseDir("./pages/docs/_docs/"));

    const categories = {};

    // Iterate through each docs page and add the category.
    Object.values(memoizedDocs.docs).forEach((d: Doc) => {
      if (!d.scope.category) {
        console.warn("no category for doc", JSON.stringify(d.scope));
        return;
      }

      // Add category to list.
      if (!categories[d.scope.category]) {
        categories[d.scope.category] = {
          title: d.scope.category,
          pages: [d.scope],
        }
      } else {
        categories[d.scope.category].pages.push(d.scope);
      }

    });

    memoizedDocs.categories = categories;

    return memoizedDocs;
  }
})()

export const getDocs = (slug: string): Doc | undefined => {
  const docs = getAllDocs();
  return docs.docs[slug];
}

const getHeadings = (content: string) => {
  // Get headers for table of contents.
  const slugify = require('slugify');
  const headings = {};
  let h2 = null; // store the current heading we're in
  let order = 0;

  (content.match(/^###? (.*)/gm) || []).forEach(heading => {
    const title = heading.replace(/^###? /, "");
    if (heading.indexOf("## ") === 0) {
      h2 = title;
      headings[title] = { title, slug: slugify(title).toLowerCase(), subheadings: [], order }
      order++
      return;
    }
    // add this subheading to the current heading list.
    headings[h2].subheadings.push({ title, slug: slugify(title) });
  });
  return headings;
}
