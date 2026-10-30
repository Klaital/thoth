import { useEffect, useState, useCallback } from "react";
import { useParams, Link, useNavigate } from "react-router-dom";
import { apiFetch } from "../api/auth";
import {
  Collection,
  CardItem,
  SealedItem,
  BookDvdItem,
  FOIL_TREATMENT_LABELS,
  CardCategory,
  CARD_CATEGORY_LABELS,
  CARD_CATEGORY_OPTIONS,
} from "../api/types";
import AppLayout from "../components/AppLayout";
import AddCardItemForm from "../components/AddCardItemForm";
import AddSealedItemForm from "../components/AddSealedItemForm";
import BulkImportForm from "../components/BulkImportForm";
import AddBookDvdItemForm from "../components/AddBookDvdItemForm";

type AnyItem = CardItem | SealedItem | BookDvdItem;

const RECENT_LIMIT = 20;

const SEALED_BULK_PLACEHOLDER = `Reality Fracture Play Booster Display
Reality Fracture Collector Booster Display
Secret Lair x The Hobbit: Smaug's Spoils Foil Edition`;

const CARD_BULK_PLACEHOLDER = `4 Lightning Bolt (M20) 141
1 Teferi, Hero of Dominaria (DOM) 207
1 Sol Ring (C21) 263 *F*`;

function itemsEndpoint(type: Collection["type"], id: string): string | null {
  switch (type) {
    case "COLLECTION_TYPE_MTG_CARDS":
    case "COLLECTION_TYPE_MTG_DECK":
      return `/v1/collections/${id}/cards`;
    case "COLLECTION_TYPE_MTG_SEALED":
      return `/v1/collections/${id}/sealed`;
    case "COLLECTION_TYPE_BOOKS":
    case "COLLECTION_TYPE_DVDS":
      return `/v1/collections/${id}/items`;
    default:
      return null;
  }
}

function sortedRecent(items: AnyItem[]): AnyItem[] {
  return [...items]
    .sort((a, b) => Number(BigInt(b.id) - BigInt(a.id)))
    .slice(0, RECENT_LIMIT);
}

function itemLabel(item: AnyItem, type: Collection["type"]): string {
  if (type === "COLLECTION_TYPE_MTG_CARDS" || type === "COLLECTION_TYPE_MTG_DECK") {
    const c = item as CardItem;
    return `${c.cardName} · ${c.setCode.toUpperCase()} #${c.collectorNumber} · ${FOIL_TREATMENT_LABELS[c.foilTreatment]}`;
  }
  if (type === "COLLECTION_TYPE_MTG_SEALED") {
    const s = item as SealedItem;
    const base = s.name || (s.setCode ? s.setCode.toUpperCase() : "");
    return base || "Sealed item";
  }
  return (item as BookDvdItem).title;
}

export default function AddItemPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [collection, setCollection] = useState<Collection | null>(null);
  const [loading, setLoading] = useState(true);
  const [recentItems, setRecentItems] = useState<AnyItem[]>([]);
  const [addCount, setAddCount] = useState(0);
  const [bulkMode, setBulkMode] = useState(false);
  const [bulkCategory, setBulkCategory] = useState<CardCategory>("CARD_CATEGORY_NONE");

  useEffect(() => {
    apiFetch(`/v1/collections/${id}`)
      .then((r) => r.json())
      .then((data) => setCollection(data.collection ?? null))
      .finally(() => setLoading(false));
  }, [id]);

  const fetchRecent = useCallback(async () => {
    if (!collection) return;
    const endpoint = itemsEndpoint(collection.type, id!);
    if (!endpoint) return;
    const r = await apiFetch(endpoint);
    if (!r.ok) return;
    const data = await r.json();
    setRecentItems(sortedRecent(data.items ?? []));
  }, [collection, id]);

  useEffect(() => {
    fetchRecent();
  }, [fetchRecent, addCount]);

  function onSuccess() {
    navigate(`/collections/${id}`);
  }

  function onItemAdded() {
    setAddCount((c) => c + 1);
  }

  if (loading) return <AppLayout><p className="loading">Loading…</p></AppLayout>;
  if (!collection) return <AppLayout><p className="error-page">Collection not found.</p></AppLayout>;

  const collectionId = Number(id);

  return (
    <AppLayout>
      <div className="page">
        <Link to={`/collections/${id}`} className="back-link">← Back to {collection.name}</Link>

        <div className="add-item-layout">
          <div className="form-card">
            <h1>Add Item</h1>
            {collection.type !== "COLLECTION_TYPE_BOOKS" && collection.type !== "COLLECTION_TYPE_DVDS" && (
              <div className="mode-toggle" role="group" aria-label="Add mode">
                <button type="button" aria-pressed={!bulkMode} onClick={() => setBulkMode(false)}>
                  Single item
                </button>
                <button type="button" aria-pressed={bulkMode} onClick={() => setBulkMode(true)}>
                  Bulk import
                </button>
              </div>
            )}
            {(collection.type === "COLLECTION_TYPE_MTG_CARDS" || collection.type === "COLLECTION_TYPE_MTG_DECK") && (
              bulkMode ? (
                <BulkImportForm
                  collectionId={collectionId}
                  endpoint="cards/bulk"
                  extraBody={{ category: bulkCategory }}
                  extraFields={
                    <div className="field">
                      <label htmlFor="bulk-category">Category for imported cards</label>
                      <select
                        id="bulk-category"
                        value={bulkCategory}
                        onChange={(e) => setBulkCategory(e.target.value as CardCategory)}
                      >
                        {CARD_CATEGORY_OPTIONS.map((c) => (
                          <option key={c} value={c}>{CARD_CATEGORY_LABELS[c]}</option>
                        ))}
                      </select>
                    </div>
                  }
                  placeholder={CARD_BULK_PLACEHOLDER}
                  help={
                    <>
                      Moxfield format: <em>&lt;qty&gt; &lt;name&gt; (&lt;set&gt;) &lt;number&gt;</em>, with an
                      optional <code>*F*</code> (foil) or <code>*E*</code> (etched) at the end. The set and collector
                      number are required. Header lines <code>Commander</code>, <code>Deck</code> and{" "}
                      <code>Sideboard</code> set the category of the cards below them (cards before the first header
                      go in Deck) and override the category picked below.
                    </>
                  }
                  onSuccess={onSuccess}
                  onItemAdded={onItemAdded}
                />
              ) : (
                <AddCardItemForm collectionId={collectionId} onDone={onSuccess} onItemAdded={onItemAdded} />
              )
            )}
            {collection.type === "COLLECTION_TYPE_MTG_SEALED" && (
              bulkMode ? (
                <BulkImportForm
                  collectionId={collectionId}
                  endpoint="sealed/bulk"
                  placeholder={SEALED_BULK_PLACEHOLDER}
                  help={
                    <>
                      Lines with "Play Booster" or "Collector Booster" are read as{" "}
                      <em>&lt;set name&gt; &lt;product&gt;</em>. Everything else is added as a Secret Lair.
                      Repeated lines increase the quantity.
                    </>
                  }
                  onSuccess={onSuccess}
                  onItemAdded={onItemAdded}
                />
              ) : (
                <AddSealedItemForm collectionId={collectionId} onSuccess={onSuccess} onItemAdded={onItemAdded} />
              )
            )}
            {(collection.type === "COLLECTION_TYPE_BOOKS" || collection.type === "COLLECTION_TYPE_DVDS") && (
              <AddBookDvdItemForm collectionId={collectionId} onSuccess={onSuccess} onItemAdded={onItemAdded} />
            )}
          </div>

          <div className="recent-items-panel">
            <h2>Recently added</h2>
            {recentItems.length === 0 ? (
              <p className="recent-empty">Nothing yet.</p>
            ) : (
              <ol className="recent-list">
                {recentItems.map((item) => (
                  <li key={item.id} className="recent-row">
                    {itemLabel(item, collection.type)}
                    {item.quantity > 1 && <span className="recent-qty">×{item.quantity}</span>}
                  </li>
                ))}
              </ol>
            )}
          </div>
        </div>
      </div>
    </AppLayout>
  );
}
