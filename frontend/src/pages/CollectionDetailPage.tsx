import { useEffect, useState } from "react";
import { useParams, Link, useNavigate } from "react-router-dom";
import { apiFetch } from "../api/auth";
import {
  Collection,
  COLLECTION_TYPE_LABELS,
  collectionTypeBadgeClass,
  CardItem,
  SealedItem,
  BookDvdItem,
  FOIL_TREATMENT_LABELS,
} from "../api/types";
import AppLayout from "../components/AppLayout";
import { useAuth } from "../context/AuthContext";

export default function CollectionDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { token } = useAuth();
  const navigate = useNavigate();
  const [collection, setCollection] = useState<Collection | null>(null);
  const [items, setItems] = useState<CardItem[] | SealedItem[] | BookDvdItem[] | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    apiFetch(`/v1/collections/${id}`)
      .then(async (r) => {
        if (!r.ok) { setError("Collection not found."); return; }
        const data = await r.json();
        const col: Collection = data.collection ?? null;
        setCollection(col);
        if (col) await fetchItems(col);
      })
      .finally(() => setLoading(false));
  }, [id]);

  async function fetchItems(col: Collection) {
    const endpoint = itemsEndpoint(col.type, id!);
    if (!endpoint) return;
    const r = await apiFetch(endpoint);
    if (!r.ok) return;
    const data = await r.json();
    setItems(data.items ?? []);
  }

  async function handleDelete() {
    if (!confirm("Delete this collection? This cannot be undone.")) return;
    setDeleting(true);
    const resp = await apiFetch(`/v1/collections/${id}`, { method: "DELETE" });
    if (resp.ok) {
      navigate("/collections");
    } else {
      setDeleting(false);
    }
  }

  const userId = token ? parseUserId(token) : null;
  const isOwner = collection && userId !== null && String(userId) === collection.ownerId;

  if (loading) return <AppLayout><p className="loading">Loading…</p></AppLayout>;
  if (error || !collection) return <AppLayout><p className="error-page">{error ?? "Collection not found."}</p></AppLayout>;

  return (
    <AppLayout>
      <div className="page">
        <Link to="/collections" className="back-link">← All collections</Link>

        <div className="detail-header">
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", gap: 16 }}>
            <h1>{collection.name}</h1>
            {isOwner && (
              <button className="btn-danger" onClick={handleDelete} disabled={deleting}>
                {deleting ? "Deleting…" : "Delete"}
              </button>
            )}
          </div>
          {collection.description && <p className="desc">{collection.description}</p>}
          <div className="meta">
            <span className={collectionTypeBadgeClass(collection.type)}>
              {COLLECTION_TYPE_LABELS[collection.type]}
            </span>
            {collection.isPublic && <span className="badge badge-public">Public</span>}
          </div>
        </div>

        <div className="page-header" style={{ marginBottom: 16 }}>
          <h2 style={{ margin: 0, fontSize: 16, fontWeight: 600 }}>
            Items{items !== null ? ` (${items.length})` : ""}
          </h2>
          {isOwner && (
            <Link to={`/collections/${id}/add`}>
              <button className="btn-sm">+ Add item</button>
            </Link>
          )}
        </div>

        {items === null || items.length === 0 ? (
          <div className="empty-state" style={{ padding: "32px 0" }}>
            <p>No items yet.{isOwner ? " Add one to get started." : ""}</p>
          </div>
        ) : (
          <ItemList collection={collection} items={items} />
        )}
      </div>
    </AppLayout>
  );
}

function ItemList({ collection, items }: { collection: Collection; items: CardItem[] | SealedItem[] | BookDvdItem[] }) {
  const type = collection.type;

  if (type === "COLLECTION_TYPE_MTG_CARDS" || type === "COLLECTION_TYPE_MTG_DECK") {
    return <CardItemList items={items as CardItem[]} />;
  }
  if (type === "COLLECTION_TYPE_MTG_SEALED") {
    return <SealedItemList items={items as SealedItem[]} />;
  }
  return <BookDvdItemList items={items as BookDvdItem[]} />;
}

function CardItemList({ items }: { items: CardItem[] }) {
  return (
    <div className="item-list">
      {items.map((item) => (
        <div key={item.id} className="item-row">
          {item.imageUri && (
            <img src={item.imageUri} alt={item.cardName} className="card-thumb" />
          )}
          <div className="item-info">
            <span className="item-name">{item.cardName}</span>
            <span className="item-meta">
              {item.setCode.toUpperCase()} #{item.collectorNumber} &middot; {FOIL_TREATMENT_LABELS[item.foilTreatment]}
            </span>
            {item.notes && <span className="item-notes">{item.notes}</span>}
          </div>
          <span className="item-qty">×{item.quantity}</span>
        </div>
      ))}
    </div>
  );
}

function SealedItemList({ items }: { items: SealedItem[] }) {
  return (
    <div className="item-list">
      {items.map((item) => (
        <div key={item.id} className="item-row">
          <div className="item-info">
            <span className="item-name">{item.setCode.toUpperCase()}</span>
            {item.description && <span className="item-meta">{item.description}</span>}
            {item.notes && <span className="item-notes">{item.notes}</span>}
          </div>
          <span className="item-qty">×{item.quantity}</span>
        </div>
      ))}
    </div>
  );
}

function BookDvdItemList({ items }: { items: BookDvdItem[] }) {
  return (
    <div className="item-list">
      {items.map((item) => (
        <div key={item.id} className="item-row">
          <div className="item-info">
            <span className="item-name">{item.title}</span>
            {item.upc && <span className="item-meta">UPC: {item.upc}</span>}
            {item.notes && <span className="item-notes">{item.notes}</span>}
          </div>
          <span className="item-qty">×{item.quantity}</span>
        </div>
      ))}
    </div>
  );
}

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

function parseUserId(token: string): number | null {
  try {
    const payload = JSON.parse(atob(token.split(".")[1]));
    return payload.uid ?? null;
  } catch {
    return null;
  }
}
