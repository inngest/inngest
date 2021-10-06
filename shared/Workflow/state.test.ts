import { detectCycles } from "./state";
import { describe, expect, it } from "@jest/globals";

describe("detectCycles", () => {
  it("works with complex cases", () => {
    const edges = [
      { outgoing: "trigger", incoming: 1 },
      { outgoing: "trigger", incoming: 2 },
      { outgoing: 1, incoming: 6 }, // 1 -> 5 -> 6
      { outgoing: 2, incoming: 3 }, // 2 -> 3 -> 4 -> 5 -> 6
      { outgoing: 3, incoming: 4 },
      { outgoing: 4, incoming: 5 },
      { outgoing: 5, incoming: 6 },
    ];
    expect(detectCycles(edges)).toEqual(false);
  });

  it("detects self referential edges", () => {
    const edges = [
      { outgoing: "trigger", incoming: 1 },
      { outgoing: 1, incoming: 1 },
    ];
    expect(detectCycles(edges)).toEqual(true);
  });

  it("detects duplicative", () => {
    const edges = [
      { outgoing: "trigger", incoming: 1 },
      { outgoing: 1, incoming: 2 },
      { outgoing: 1, incoming: 2 },
    ];
    expect(detectCycles(edges)).toEqual(true);
  });

  it("detects cycles with complex cases", () => {
    const edges = [
      { outgoing: "trigger", incoming: 1 },
      { outgoing: "trigger", incoming: 2 },
      { outgoing: 1, incoming: 3 }, // 1 -> 3 -> 4 -> 5
      { outgoing: 2, incoming: 4 }, // 2 -> 4 -> 5
      { outgoing: 3, incoming: 4 },
      { outgoing: 4, incoming: 5 }, // branch
      { outgoing: 5, incoming: 6 },
      { outgoing: 5, incoming: 7 },
      { outgoing: 7, incoming: 1 },
    ];
    expect(detectCycles(edges)).toEqual(true);
  });
});
