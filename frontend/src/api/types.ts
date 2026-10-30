export type FoilTreatment =
  | "FOIL_TREATMENT_NONFOIL"
  | "FOIL_TREATMENT_FOIL"
  | "FOIL_TREATMENT_ETCHED";

export const FOIL_TREATMENT_LABELS: Record<FoilTreatment, string> = {
  FOIL_TREATMENT_NONFOIL: "Non-foil",
  FOIL_TREATMENT_FOIL:    "Foil",
  FOIL_TREATMENT_ETCHED:  "Etched",
};

export type CardCategory =
  | "CARD_CATEGORY_NONE"
  | "CARD_CATEGORY_COMMANDER"
  | "CARD_CATEGORY_DECK"
  | "CARD_CATEGORY_SIDEBOARD";

export const CARD_CATEGORY_LABELS: Record<CardCategory, string> = {
  CARD_CATEGORY_NONE:      "None",
  CARD_CATEGORY_COMMANDER: "Commander",
  CARD_CATEGORY_DECK:      "Deck",
  CARD_CATEGORY_SIDEBOARD: "Sideboard",
};

export const CARD_CATEGORY_OPTIONS: CardCategory[] = [
  "CARD_CATEGORY_NONE",
  "CARD_CATEGORY_COMMANDER",
  "CARD_CATEGORY_DECK",
  "CARD_CATEGORY_SIDEBOARD",
];

export interface CardItem {
  id: string;
  collectionId: string;
  scryfallCardId: string;
  foilTreatment: FoilTreatment;
  category: CardCategory;
  quantity: number;
  notes: string;
  cardName: string;
  setCode: string;
  collectorNumber: string;
  imageUri: string;
  pricesJson: string;
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
  // Card imports only: the name parsed from the line.
  name?: string;
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
  | "COLLECTION_TYPE_MTG_DECK"
  | "COLLECTION_TYPE_BOOKS"
  | "COLLECTION_TYPE_DVDS"
  | "COLLECTION_TYPE_UNSPECIFIED";

export interface CollectionCommander {
  cardName: string;
  imageUri: string;
}

export interface Collection {
  id: string;
  ownerId: string;
  name: string;
  description: string;
  type: CollectionType;
  isPublic: boolean;
  commanders?: CollectionCommander[];
}

export const COLLECTION_TYPE_LABELS: Record<CollectionType, string> = {
  COLLECTION_TYPE_MTG_CARDS:  "MtG Cards",
  COLLECTION_TYPE_MTG_SEALED: "MtG Sealed",
  COLLECTION_TYPE_MTG_DECK:   "MtG Deck",
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
