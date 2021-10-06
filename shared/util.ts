import isEqual from "react-fast-compare";

export const slugify = (str: string) => {
  return encodeURIComponent(str.toLowerCase().replace(" ", "-"));
}

export const displayCase = (str: string): string => {
  return str.split("_").map(titleCase).join(" ");
};

export const snakeCase = (str: string): string => {
  const words = str.split("_");
  return [words[0]].concat(words.splice(1).map(titleCase)).join("");
};

export const titleCase = (str: string): string => {
  if (str.length === 0) {
    return str;
  }
  const down = str.toLowerCase();
  if (down === "id") {
    return "ID";
  }
  return `${str.substr(0, 1).toUpperCase()}${down.substr(1)}`.replace("_", " ");
};

export const toggle = <T extends any>(input: Array<T>, item: T): Array<T> => {
  if (!input) {
    return [] as Array<T>;
  }

  // Remove the __typename fields here;  we only care about values.  This is specific
  // to tasks:  we assign __typename of "Staff" but the API responds with __typename of "User".
  let a = item;
  if (typeof a === "object") {
    a = Object.assign({}, item, { __typename: null });
  }

  // these may be different objects with the same value under the hood,
  // as the search component hits GQL and creates a new map every time.
  //
  // Therefore, use react-fast-equal to find our index
  const idx = input.findIndex((b) => {
    if (typeof b === "object") {
      return isEqual(a, Object.assign({}, b, { __typename: null }));
    }
    return isEqual(a, b);
  });

  if (idx > -1 && input.length > 1) {
    const copy = input.slice(0);
    copy.splice(idx, 1);
    return copy;
  }

  if (idx > -1 && input.length === 1) {
    return [] as Array<T>;
  }

  return input.concat([item]);
};
