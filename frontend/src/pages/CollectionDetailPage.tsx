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

  async function handleDeleteCollection() {
    if (!confirm("Delete this collection? This cannot be undone.")) return;
    setDeleting(true);
    const resp = await apiFetch(`/v1/collections/${id}`, { method: "DELETE" });
    if (resp.ok) {
      navigate("/collections");
    } else {
      setDeleting(false);
    }
  }

  async function handleRemoveItem(itemId: string) {
    if (!collection) return;
    const endpoint = removeEndpoint(collection.type, itemId);
    if (!endpoint) return;
    const resp = await apiFetch(endpoint, { method: "DELETE" });
    if (resp.ok) {
      setItems((prev) => prev ? (prev as Array<{ id: string }>).filter((i) => i.id !== itemId) as typeof prev : prev);
    }
  }

  async function handleQuantityChange(itemId: string, delta: 1 | -1) {
    if (!collection) return;
    const endpoint = updateEndpoint(collection.type, itemId);
    if (!endpoint) return;

    // Build body, preserving non-quantity fields, and optimistically update state.
    const current = (items as Array<{ id: string; quantity: number }>).find((i) => i.id === itemId);
    if (!current) return;
    const newQty = Math.max(1, current.quantity + delta);
    if (newQty === current.quantity) return;

    // Optimistic update.
    setItems((prev) =>
      prev ? (prev as Array<{ id: string; quantity: number }>).map((i) =>
        i.id === itemId ? { ...i, quantity: newQty } : i
      ) as typeof prev : prev
    );

    const body = buildUpdateBody(collection.type, items!, itemId, newQty);
    const resp = await apiFetch(endpoint, { method: "PATCH", body: JSON.stringify(body) });
    if (resp.ok) {
      const data = await resp.json();
      const updated = data.item;
      if (updated) {
        setItems((prev) =>
          prev ? (prev as Array<{ id: string }>).map((i) =>
            i.id === itemId ? { ...i, ...updated } : i
          ) as typeof prev : prev
        );
      }
    } else {
      // Revert on failure.
      setItems((prev) =>
        prev ? (prev as Array<{ id: string; quantity: number }>).map((i) =>
          i.id === itemId ? { ...i, quantity: current.quantity } : i
        ) as typeof prev : prev
      );
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
              <button className="btn-danger" onClick={handleDeleteCollection} disabled={deleting}>
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
          <ItemList
            collection={collection}
            items={items}
            isOwner={!!isOwner}
            onRemove={handleRemoveItem}
            onQuantityChange={handleQuantityChange}
          />
        )}
      </div>
    </AppLayout>
  );
}

interface ItemListProps {
  collection: Collection;
  items: CardItem[] | SealedItem[] | BookDvdItem[];
  isOwner: boolean;
  onRemove: (id: string) => void;
  onQuantityChange: (id: string, delta: 1 | -1) => void;
}

function ItemList({ collection, items, isOwner, onRemove, onQuantityChange }: ItemListProps) {
  const type = collection.type;
  if (type === "COLLECTION_TYPE_MTG_CARDS" || type === "COLLECTION_TYPE_MTG_DECK") {
    return <CardItemList items={items as CardItem[]} isOwner={isOwner} onRemove={onRemove} onQuantityChange={onQuantityChange} />;
  }
  if (type === "COLLECTION_TYPE_MTG_SEALED") {
    return <SealedItemList items={items as SealedItem[]} isOwner={isOwner} onRemove={onRemove} onQuantityChange={onQuantityChange} />;
  }
  return <BookDvdItemList items={items as BookDvdItem[]} isOwner={isOwner} onRemove={onRemove} onQuantityChange={onQuantityChange} />;
}

interface RowProps {
  isOwner: boolean;
  onRemove: (id: string) => void;
  onQuantityChange: (id: string, delta: 1 | -1) => void;
}

function RemoveButton({ onRemove }: { onRemove: () => void }) {
  return (
    <button
      className="btn-ghost"
      onClick={onRemove}
      title="Remove from collection"
      style={{ color: "var(--danger)", flexShrink: 0, padding: "4px 8px", fontSize: 13 }}
    >
      ✕
    </button>
  );
}

function QuantityStepper({ id, quantity, isOwner, onQuantityChange }: {
  id: string;
  quantity: number;
  isOwner: boolean;
  onQuantityChange: (id: string, delta: 1 | -1) => void;
}) {
  if (!isOwner) {
    return <span className="item-qty">×{quantity}</span>;
  }
  return (
    <div className="qty-stepper">
      <button
        className="btn-ghost"
        onClick={() => onQuantityChange(id, -1)}
        disabled={quantity <= 1}
        aria-label="Decrease quantity"
      >−</button>
      <span className="item-qty">{quantity}</span>
      <button
        className="btn-ghost"
        onClick={() => onQuantityChange(id, 1)}
        aria-label="Increase quantity"
      >+</button>
    </div>
  );
}

function unitPrice(item: CardItem): number | null {
  if (!item.pricesJson) return null;
  try {
    const p: Record<string, string | null> = JSON.parse(item.pricesJson);
    const key =
      item.foilTreatment === "FOIL_TREATMENT_FOIL"   ? "usd_foil"   :
      item.foilTreatment === "FOIL_TREATMENT_ETCHED" ? "usd_etched" : "usd";
    const v = p[key];
    if (!v) return null;
    const n = parseFloat(v);
    return isNaN(n) ? null : n;
  } catch {
    return null;
  }
}

function fmt(n: number): string {
  return "$" + n.toFixed(2);
}

function CardItemList({ items, isOwner, onRemove, onQuantityChange }: { items: CardItem[] } & RowProps) {
  let collectionTotal = 0;
  let hasAnyPrice = false;

  for (const item of items) {
    const p = unitPrice(item);
    if (p !== null) { collectionTotal += p * item.quantity; hasAnyPrice = true; }
  }

  return (
    <>
      {hasAnyPrice && (
        <div className="collection-total">
          Collection value: <strong>{fmt(collectionTotal)}</strong>
        </div>
      )}
      <div className="item-list">
        {items.map((item) => {
          const unit = unitPrice(item);
          const subtotal = unit !== null ? unit * item.quantity : null;
          return (
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
              {unit !== null && (
                <div className="item-price">
                  <span className="price-unit">{fmt(unit)}</span>
                  {item.quantity > 1 && subtotal !== null && (
                    <span className="price-subtotal">{fmt(subtotal)}</span>
                  )}
                </div>
              )}
              <QuantityStepper id={item.id} quantity={item.quantity} isOwner={isOwner} onQuantityChange={onQuantityChange} />
              {isOwner && <RemoveButton onRemove={() => onRemove(item.id)} />}
            </div>
          );
        })}
      </div>
    </>
  );
}

function SealedItemList({ items, isOwner, onRemove, onQuantityChange }: { items: SealedItem[] } & RowProps) {
  return (
    <div className="item-list">
      {items.map((item) => (
        <div key={item.id} className="item-row">
          <div className="item-info">
            <span className="item-name">{item.setCode.toUpperCase()}</span>
            {item.description && <span className="item-meta">{item.description}</span>}
            {item.notes && <span className="item-notes">{item.notes}</span>}
          </div>
          <QuantityStepper id={item.id} quantity={item.quantity} isOwner={isOwner} onQuantityChange={onQuantityChange} />
          {isOwner && <RemoveButton onRemove={() => onRemove(item.id)} />}
        </div>
      ))}
    </div>
  );
}

function BookDvdItemList({ items, isOwner, onRemove, onQuantityChange }: { items: BookDvdItem[] } & RowProps) {
  return (
    <div className="item-list">
      {items.map((item) => (
        <div key={item.id} className="item-row">
          <div className="item-info">
            <span className="item-name">{item.title}</span>
            {item.upc && <span className="item-meta">UPC: {item.upc}</span>}
            {item.notes && <span className="item-notes">{item.notes}</span>}
          </div>
          <QuantityStepper id={item.id} quantity={item.quantity} isOwner={isOwner} onQuantityChange={onQuantityChange} />
          {isOwner && <RemoveButton onRemove={() => onRemove(item.id)} />}
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

function removeEndpoint(type: Collection["type"], itemId: string): string | null {
  switch (type) {
    case "COLLECTION_TYPE_MTG_CARDS":
    case "COLLECTION_TYPE_MTG_DECK":
      return `/v1/cards/${itemId}`;
    case "COLLECTION_TYPE_MTG_SEALED":
      return `/v1/sealed/${itemId}`;
    case "COLLECTION_TYPE_BOOKS":
    case "COLLECTION_TYPE_DVDS":
      return `/v1/items/${itemId}`;
    default:
      return null;
  }
}

function updateEndpoint(type: Collection["type"], itemId: string): string | null {
  switch (type) {
    case "COLLECTION_TYPE_MTG_CARDS":
    case "COLLECTION_TYPE_MTG_DECK":
      return `/v1/cards/${itemId}`;
    case "COLLECTION_TYPE_MTG_SEALED":
      return `/v1/sealed/${itemId}`;
    case "COLLECTION_TYPE_BOOKS":
    case "COLLECTION_TYPE_DVDS":
      return `/v1/items/${itemId}`;
    default:
      return null;
  }
}

function buildUpdateBody(
  type: Collection["type"],
  items: CardItem[] | SealedItem[] | BookDvdItem[],
  itemId: string,
  newQty: number,
): Record<string, unknown> {
  if (type === "COLLECTION_TYPE_MTG_CARDS" || type === "COLLECTION_TYPE_MTG_DECK") {
    const item = (items as CardItem[]).find((i) => i.id === itemId)!;
    return { quantity: newQty, notes: item.notes };
  }
  if (type === "COLLECTION_TYPE_MTG_SEALED") {
    const item = (items as SealedItem[]).find((i) => i.id === itemId)!;
    return { description: item.description, quantity: newQty, notes: item.notes };
  }
  const item = (items as BookDvdItem[]).find((i) => i.id === itemId)!;
  return { title: item.title, quantity: newQty, notes: item.notes };
}

function parseUserId(token: string): number | null {
  try {
    const payload = JSON.parse(atob(token.split(".")[1]));
    return payload.uid ?? null;
  } catch {
    return null;
  }
}
