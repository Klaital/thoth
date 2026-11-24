import { describe, expect, test } from "vitest";
import {
  describeLocations,
  describePrinting,
  groupCheckoutsByCollection,
  initialPicks,
  pullList,
  validatePicks,
} from "./checkout";
import { DeckCheckout, DeckOwnership, OwnedCopy } from "./types";

function copy(id: string, collectionId: string, collectionName: string, extra: Partial<OwnedCopy> = {}): OwnedCopy {
  return {
    collectionCardId: id,
    collectionId,
    collectionName,
    scryfallCardId: `p-${id}`,
    setCode: "mh2",
    collectorNumber: "259",
    foilTreatment: "FOIL_TREATMENT_NONFOIL",
    language: "en",
    quantity: 1,
    free: 1,
    ...extra,
  };
}

// The gateway omits zero counts, so some fields are missing on purpose.
const ownership: DeckOwnership = {
  cards: [
    {
      cardName: "Sol Ring",
      needed: 1,
      free: 2,
      copies: [copy("s1", "2", "Box B", { quantity: 2, free: 2, suggestedQuantity: 1 })],
    },
    {
      cardName: "City of Brass",
      needed: 3,
      checkedOutHere: 1,
      free: 3,
      copies: [
        copy("c1", "1", "Box A", { quantity: 2, free: 1, checkedOutHere: 1, suggestedQuantity: 1 }),
        copy("c2", "2", "Box B", { suggestedQuantity: 1 }),
        copy("c3", "1", "Box A", { foilTreatment: "FOIL_TREATMENT_FOIL" }),
      ],
    },
    { cardName: "Swords to Plowshares", needed: 1, missing: 1 },
  ],
  totalMissing: 1,
};

describe("check-out helpers", () => {
  test("initial picks follow the server's suggestions", () => {
    expect(initialPicks(ownership)).toEqual({ s1: 1, c1: 1, c2: 1 });
  });

  test("pull list goes box by box, cards sorted by name", () => {
    const groups = pullList(ownership, initialPicks(ownership));
    expect(groups.map((g) => [g.collectionName, g.lines.map((l) => `${l.quantity} ${l.cardName}`)])).toEqual([
      ["Box A", ["1 City of Brass"]],
      ["Box B", ["1 City of Brass", "1 Sol Ring"]],
    ]);
  });

  test("validation catches more than is free and more than the deck needs", () => {
    expect(validatePicks(ownership, initialPicks(ownership))).toEqual([]);
    expect(validatePicks(ownership, { c1: 2 })).toEqual(["City of Brass: only 1 free in Box A."]);
    expect(validatePicks(ownership, { c1: 1, c2: 1, c3: 1 })).toEqual([
      "City of Brass: 3 picked, but the deck only needs 2 more.",
    ]);
  });

  test("checkouts group by collection in server order", () => {
    const co = (id: string, collectionId: string, collectionName: string): DeckCheckout => ({
      collectionCardId: id, collectionId, collectionName, cardName: id, scryfallCardId: id,
      setCode: "c21", collectorNumber: "1", foilTreatment: "FOIL_TREATMENT_NONFOIL", language: "en", quantity: 1,
    });
    const groups = groupCheckoutsByCollection([co("a", "1", "Box A"), co("b", "1", "Box A"), co("c", "2", "Box B")]);
    expect(groups.map((g) => [g.collectionName, g.lines.length])).toEqual([["Box A", 2], ["Box B", 1]]);
  });

  test("describes printings and locations", () => {
    expect(describePrinting({ setCode: "mh2", collectorNumber: "259", foilTreatment: "FOIL_TREATMENT_FOIL", language: "ja" }))
      .toBe("MH2 #259 · Foil · Japanese");
    expect(describeLocations(ownership.cards![1])).toBe("Box A ×2 · Box B ×1 · checked out here ×1");
  });
});
