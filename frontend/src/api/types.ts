export type FoilTreatment =
  | "FOIL_TREATMENT_NONFOIL"
  | "FOIL_TREATMENT_FOIL"
  | "FOIL_TREATMENT_ETCHED";

export const FOIL_TREATMENT_LABELS: Record<FoilTreatment, string> = {
  FOIL_TREATMENT_NONFOIL: "Non-foil",
  FOIL_TREATMENT_FOIL:    "Foil",
  FOIL_TREATMENT_ETCHED:  "Etched",
};

// Card languages, using Scryfall's language codes. The backend accepts any
// 2-3 letter code; these are the ones Magic has been printed in.
export const LANGUAGE_OPTIONS: { code: string; label: string }[] = [
  { code: "en",  label: "English" },
  { code: "ja",  label: "Japanese" },
  { code: "de",  label: "German" },
  { code: "fr",  label: "French" },
  { code: "it",  label: "Italian" },
  { code: "es",  label: "Spanish" },
  { code: "pt",  label: "Portuguese" },
  { code: "ko",  label: "Korean" },
  { code: "ru",  label: "Russian" },
  { code: "zhs", label: "Chinese (Simplified)" },
  { code: "zht", label: "Chinese (Traditional)" },
  { code: "he",  label: "Hebrew" },
  { code: "la",  label: "Latin" },
  { code: "grc", label: "Ancient Greek" },
  { code: "ar",  label: "Arabic" },
  { code: "sa",  label: "Sanskrit" },
  { code: "ph",  label: "Phyrexian" },
];

export function languageLabel(code: string): string {
  return LANGUAGE_OPTIONS.find((l) => l.code === code)?.label ?? code.toUpperCase();
}

export interface CardItem {
  id: string;
  collectionId: string;
  scryfallCardId: string;
  foilTreatment: FoilTreatment;
  language: string;
  quantity: number;
  notes: string;
  cardName: string;
  setCode: string;
  collectorNumber: string;
  imageUri: string;
  scryfallUri: string;
  pricesJson: string;
  // Copies checked out to decks; only sent to the collection's owner.
  checkouts?: CardCheckout[];
}

export interface CardCheckout {
  deckId: string;
  deckName: string;
  quantity: number;
}

export type SealedItemType =
  | "SEALED_ITEM_TYPE_UNSPECIFIED"
  | "SEALED_ITEM_TYPE_SECRET_LAIR"
  | "SEALED_ITEM_TYPE_PLAY_BOOSTER"
  | "SEALED_ITEM_TYPE_PLAY_BOOSTER_BOX"
  | "SEALED_ITEM_TYPE_COLLECTOR_BOOSTER"
  | "SEALED_ITEM_TYPE_COLLECTOR_BOOSTER_BOX"
  | "SEALED_ITEM_TYPE_BUNDLE"
  | "SEALED_ITEM_TYPE_DRAFT_NIGHT"
  | "SEALED_ITEM_TYPE_DRAFT_BOOSTER"
  | "SEALED_ITEM_TYPE_DRAFT_BOOSTER_BOX"
  | "SEALED_ITEM_TYPE_SET_BOOSTER"
  | "SEALED_ITEM_TYPE_SET_BOOSTER_BOX"
  | "SEALED_ITEM_TYPE_JUMPSTART_BOOSTER"
  | "SEALED_ITEM_TYPE_JUMPSTART_BOOSTER_BOX"
  | "SEALED_ITEM_TYPE_COMMANDER_PRECON"
  | "SEALED_ITEM_TYPE_PRERELEASE_PACK";

export const SEALED_ITEM_TYPE_LABELS: Record<SealedItemType, string> = {
  SEALED_ITEM_TYPE_UNSPECIFIED:            "Unknown",
  SEALED_ITEM_TYPE_SECRET_LAIR:            "Secret Lair",
  SEALED_ITEM_TYPE_PLAY_BOOSTER:           "Play Booster",
  SEALED_ITEM_TYPE_PLAY_BOOSTER_BOX:       "Play Booster Box",
  SEALED_ITEM_TYPE_COLLECTOR_BOOSTER:      "Collector Booster",
  SEALED_ITEM_TYPE_COLLECTOR_BOOSTER_BOX:  "Collector Booster Box",
  SEALED_ITEM_TYPE_BUNDLE:                 "Bundle",
  SEALED_ITEM_TYPE_DRAFT_NIGHT:            "Draft Night",
  SEALED_ITEM_TYPE_DRAFT_BOOSTER:          "Draft Booster",
  SEALED_ITEM_TYPE_DRAFT_BOOSTER_BOX:      "Draft Booster Box",
  SEALED_ITEM_TYPE_SET_BOOSTER:            "Set Booster",
  SEALED_ITEM_TYPE_SET_BOOSTER_BOX:        "Set Booster Box",
  SEALED_ITEM_TYPE_JUMPSTART_BOOSTER:      "Jumpstart Booster",
  SEALED_ITEM_TYPE_JUMPSTART_BOOSTER_BOX:  "Jumpstart Booster Box",
  SEALED_ITEM_TYPE_COMMANDER_PRECON:       "Commander Precon",
  SEALED_ITEM_TYPE_PRERELEASE_PACK:        "Prerelease Pack",
};

export const SEALED_ITEM_TYPE_OPTIONS: SealedItemType[] = [
  "SEALED_ITEM_TYPE_PLAY_BOOSTER_BOX",
  "SEALED_ITEM_TYPE_COLLECTOR_BOOSTER_BOX",
  "SEALED_ITEM_TYPE_PLAY_BOOSTER",
  "SEALED_ITEM_TYPE_COLLECTOR_BOOSTER",
  "SEALED_ITEM_TYPE_BUNDLE",
  "SEALED_ITEM_TYPE_DRAFT_BOOSTER_BOX",
  "SEALED_ITEM_TYPE_DRAFT_BOOSTER",
  "SEALED_ITEM_TYPE_SET_BOOSTER_BOX",
  "SEALED_ITEM_TYPE_SET_BOOSTER",
  "SEALED_ITEM_TYPE_JUMPSTART_BOOSTER_BOX",
  "SEALED_ITEM_TYPE_JUMPSTART_BOOSTER",
  "SEALED_ITEM_TYPE_COMMANDER_PRECON",
  "SEALED_ITEM_TYPE_PRERELEASE_PACK",
  "SEALED_ITEM_TYPE_SECRET_LAIR",
  "SEALED_ITEM_TYPE_DRAFT_NIGHT",
];

export interface SealedItem {
  id: string;
  collectionId: string;
  itemType: SealedItemType;
  setCode?: string;
  name?: string;
  quantity: number;
  notes: string;
}

export interface BulkImportLineResult<T> {
  lineNumber: number;
  line: string;
  item?: T;
  // Deck imports return the card here instead of item.
  card?: T;
  // Card imports only: the name parsed from the line.
  name?: string;
  warning?: string;
  error?: string;
}

// Progress of one line of a bulk import, as shown while it runs.
export interface BulkImportEntry {
  lineNumber: number;
  line: string;
  label: string;
  quantity: number;
  status: "pending" | "added" | "failed";
  item?: unknown;
  warning?: string;
  error?: string;
}

export interface LookupUPCResponse {
  setName: string;
  setCode: string;
  itemType: SealedItemType;
}

export interface BookDvdItem {
  id: string;
  collectionId: string;
  upc: string;
  title: string;
  quantity: number;
  notes: string;
}

export type CollectionType =
  | "COLLECTION_TYPE_MTG_CARDS"
  | "COLLECTION_TYPE_MTG_SEALED"
  | "COLLECTION_TYPE_BOOKS"
  | "COLLECTION_TYPE_DVDS"
  | "COLLECTION_TYPE_UNSPECIFIED";

export interface Collection {
  id: string;
  ownerId: string;
  name: string;
  description: string;
  type: CollectionType;
  isPublic: boolean;
  // Each user has one default card collection; it can't be deleted.
  isDefault?: boolean;
}

export const COLLECTION_TYPE_LABELS: Record<CollectionType, string> = {
  COLLECTION_TYPE_MTG_CARDS:  "MtG Cards",
  COLLECTION_TYPE_MTG_SEALED: "MtG Sealed",
  COLLECTION_TYPE_BOOKS:      "Books",
  COLLECTION_TYPE_DVDS:       "DVDs",
  COLLECTION_TYPE_UNSPECIFIED: "Unknown",
};

export function collectionTypeBadgeClass(type: CollectionType): string {
  if (type.startsWith("COLLECTION_TYPE_MTG")) return "badge badge-mtg";
  if (type === "COLLECTION_TYPE_BOOKS") return "badge badge-books";
  if (type === "COLLECTION_TYPE_DVDS")  return "badge badge-dvds";
  return "badge";
}

export type DeckBoard =
  | "DECK_BOARD_COMMANDER"
  | "DECK_BOARD_MAIN"
  | "DECK_BOARD_SIDEBOARD";

export const DECK_BOARD_LABELS: Record<DeckBoard, string> = {
  DECK_BOARD_COMMANDER: "Commander",
  DECK_BOARD_MAIN:      "Main",
  DECK_BOARD_SIDEBOARD: "Sideboard",
};

// Also the display order on the deck page.
export const DECK_BOARD_OPTIONS: DeckBoard[] = [
  "DECK_BOARD_COMMANDER",
  "DECK_BOARD_MAIN",
  "DECK_BOARD_SIDEBOARD",
];

export interface DeckCommander {
  cardName: string;
  imageUri: string;
}

export interface Deck {
  id: string;
  ownerId: string;
  name: string;
  notes: string;
  isPublic: boolean;
  updatedAt?: string;
  commanders?: DeckCommander[];
}

export interface DeckCard {
  id: string;
  deckId: string;
  // Identifies the card; any printing of this name satisfies the deck.
  cardName: string;
  board: DeckBoard;
  quantity: number;
  preferredScryfallCardId?: string;
  preferredSetCode?: string;
  preferredCollectorNumber?: string;
  imageUri?: string;
  scryfallUri?: string;
}

// Deck ownership and check-out (TH-16, TH-17). The gateway leaves out
// zero-valued numbers, so counts are optional; read them with `?? 0`.

// One owned copy (a collection card row) of a card the deck lists.
export interface OwnedCopy {
  collectionCardId: string;
  collectionId: string;
  collectionName: string;
  scryfallCardId: string;
  setCode: string;
  collectorNumber: string;
  imageUri?: string;
  foilTreatment: FoilTreatment;
  language: string;
  quantity?: number;
  checkedOutHere?: number;
  checkedOutElsewhere?: number;
  // Copies still in the collection.
  free?: number;
  unitPriceUsd?: string;
  suggestedQuantity?: number;
}

export interface DeckCardOwnership {
  cardName: string;
  // Summed across boards; 0 when the name left the deck but copies are still
  // checked out to it.
  needed?: number;
  checkedOutHere?: number;
  free?: number;
  checkedOutElsewhere?: number;
  missing?: number;
  copies?: OwnedCopy[];
}

export interface DeckOwnership {
  cards?: DeckCardOwnership[];
  totalMissing?: number;
  totalCheckedOut?: number;
}

export interface DeckCheckout {
  collectionCardId: string;
  collectionId: string;
  collectionName: string;
  cardName: string;
  scryfallCardId: string;
  setCode: string;
  collectorNumber: string;
  imageUri?: string;
  foilTreatment: FoilTreatment;
  language: string;
  quantity: number;
  checkedOutAt?: string;
}
