import {
  DeckCardOwnership,
  DeckCheckout,
  DeckOwnership,
  FOIL_TREATMENT_LABELS,
  FoilTreatment,
  OwnedCopy,
  languageLabel,
} from "./types";

// Copies to pull for a check-out, keyed by collection card id.
export type PickQuantities = Record<string, number>;

// Copies the deck still needs pulled for this card.
export function stillNeeded(card: DeckCardOwnership): number {
  return Math.max(0, (card.needed ?? 0) - (card.checkedOutHere ?? 0));
}

export function initialPicks(ownership: DeckOwnership): PickQuantities {
  const picks: PickQuantities = {};
  for (const card of ownership.cards ?? []) {
    for (const copy of card.copies ?? []) {
      if ((copy.suggestedQuantity ?? 0) > 0) picks[copy.collectionCardId] = copy.suggestedQuantity!;
    }
  }
  return picks;
}

export function pickedFor(card: DeckCardOwnership, picks: PickQuantities): number {
  return (card.copies ?? []).reduce((n, c) => n + (picks[c.collectionCardId] ?? 0), 0);
}

// Problems that would make the check-out fail or pull more than the deck
// needs, one message per card.
export function validatePicks(ownership: DeckOwnership, picks: PickQuantities): string[] {
  const problems: string[] = [];
  for (const card of ownership.cards ?? []) {
    const over = (card.copies ?? []).find((c) => (picks[c.collectionCardId] ?? 0) > (c.free ?? 0));
    if (over) {
      problems.push(`${card.cardName}: only ${over.free ?? 0} free in ${over.collectionName}.`);
      continue;
    }
    const picked = pickedFor(card, picks);
    const need = stillNeeded(card);
    if (picked > need) problems.push(`${card.cardName}: ${picked} picked, but the deck only needs ${need} more.`);
  }
  return problems;
}

export interface PullLine {
  copy: OwnedCopy;
  cardName: string;
  quantity: number;
}

export interface CollectionGroup<T> {
  collectionId: string;
  collectionName: string;
  lines: T[];
}

// The cards to pull, box by box, sorted by collection then card name.
export function pullList(ownership: DeckOwnership, picks: PickQuantities): CollectionGroup<PullLine>[] {
  const lines: PullLine[] = [];
  for (const card of ownership.cards ?? []) {
    for (const copy of card.copies ?? []) {
      const quantity = picks[copy.collectionCardId] ?? 0;
      if (quantity > 0) lines.push({ copy, cardName: card.cardName, quantity });
    }
  }
  lines.sort((a, b) =>
    a.copy.collectionName.localeCompare(b.copy.collectionName) ||
    a.copy.collectionId.localeCompare(b.copy.collectionId) ||
    a.cardName.localeCompare(b.cardName));
  return groupByCollection(lines, (l) => l.copy);
}

// Checkouts come from the server ordered by collection; keep that order.
export function groupCheckoutsByCollection(checkouts: DeckCheckout[]): CollectionGroup<DeckCheckout>[] {
  return groupByCollection(checkouts, (c) => c);
}

function groupByCollection<T>(
  items: T[],
  where: (item: T) => { collectionId: string; collectionName: string },
): CollectionGroup<T>[] {
  const groups: CollectionGroup<T>[] = [];
  for (const item of items) {
    const { collectionId, collectionName } = where(item);
    const last = groups[groups.length - 1];
    if (last && last.collectionId === collectionId) last.lines.push(item);
    else groups.push({ collectionId, collectionName, lines: [item] });
  }
  return groups;
}

// "MH2 #259 · Foil · Japanese": what to look for in the box.
export function describePrinting(p: {
  setCode: string;
  collectorNumber: string;
  foilTreatment: FoilTreatment;
  language: string;
}): string {
  const parts = [`${p.setCode.toUpperCase()} #${p.collectorNumber}`];
  if (p.foilTreatment && p.foilTreatment !== "FOIL_TREATMENT_NONFOIL") parts.push(FOIL_TREATMENT_LABELS[p.foilTreatment]);
  if (p.language && p.language !== "en") parts.push(languageLabel(p.language));
  return parts.join(" · ");
}

// "Box A ×2 · checked out here ×1 · in other decks ×1" for a deck card row.
export function describeLocations(card: DeckCardOwnership): string {
  const byCollection = new Map<string, number>();
  for (const c of card.copies ?? []) {
    if ((c.free ?? 0) > 0) byCollection.set(c.collectionName, (byCollection.get(c.collectionName) ?? 0) + c.free!);
  }
  const parts = [...byCollection].map(([name, n]) => `${name} ×${n}`);
  if ((card.checkedOutHere ?? 0) > 0) parts.push(`checked out here ×${card.checkedOutHere}`);
  if ((card.checkedOutElsewhere ?? 0) > 0) parts.push(`in other decks ×${card.checkedOutElsewhere}`);
  return parts.join(" · ");
}
