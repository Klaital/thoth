export type FoilTreatment =
  | "FOIL_TREATMENT_NONFOIL"
  | "FOIL_TREATMENT_FOIL"
  | "FOIL_TREATMENT_ETCHED";

export const FOIL_TREATMENT_LABELS: Record<FoilTreatment, string> = {
  FOIL_TREATMENT_NONFOIL: "Non-foil",
  FOIL_TREATMENT_FOIL:    "Foil",
  FOIL_TREATMENT_ETCHED:  "Etched",
};

export interface CardItem {
  id: string;
  collectionId: string;
  scryfallCardId: string;
  foilTreatment: FoilTreatment;
  quantity: number;
  notes: string;
  cardName: string;
  setCode: string;
  collectorNumber: string;
  imageUri: string;
  pricesJson: string;
}

export interface SealedItem {
  id: string;
  collectionId: string;
  setCode: string;
  description: string;
  quantity: number;
  notes: string;
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

export interface Collection {
  id: string;
  ownerId: string;
  name: string;
  description: string;
  type: CollectionType;
  isPublic: boolean;
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
