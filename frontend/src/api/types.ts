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
