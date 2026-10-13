import { useEffect, useState, useCallback } from "react";
import { useParams, Link, useNavigate } from "react-router-dom";
import { apiFetch } from "../api/auth";
import {
  Collection,
  CardItem,
  SealedItem,
  BookDvdItem,
  FOIL_TREATMENT_LABELS,
} from "../api/types";
import AppLayout from "../components/AppLayout";
import AddCardItemForm from "../components/AddCardItemForm";
import AddSealedItemForm from "../components/AddSealedItemForm";
import AddBookDvdItemForm from "../components/AddBookDvdItemForm";

type AnyItem = CardItem | SealedItem | BookDvdItem;

const RECENT_LIMIT = 20;

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
    return s.description ? `${s.setCode.toUpperCase()} — ${s.description}` : s.setCode.toUpperCase();
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
            {(collection.type === "COLLECTION_TYPE_MTG_CARDS" || collection.type === "COLLECTION_TYPE_MTG_DECK") && (
              <AddCardItemForm collectionId={collectionId} onDone={onSuccess} onItemAdded={onItemAdded} />
            )}
            {collection.type === "COLLECTION_TYPE_MTG_SEALED" && (
              <AddSealedItemForm collectionId={collectionId} onSuccess={onSuccess} onItemAdded={onItemAdded} />
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
