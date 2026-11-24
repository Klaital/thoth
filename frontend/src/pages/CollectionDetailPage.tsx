import { useEffect, useState } from "react";
import { useParams, Link, useNavigate } from "react-router-dom";
import { apiFetch, parseUserId } from "../api/auth";
import {
  Collection,
  COLLECTION_TYPE_LABELS,
  collectionTypeBadgeClass,
  CardItem,
  LANGUAGE_OPTIONS,
  languageLabel,
  SealedItem,
  BookDvdItem,
  FOIL_TREATMENT_LABELS,
  SEALED_ITEM_TYPE_LABELS,
} from "../api/types";
import AppLayout from "../components/AppLayout";
import QuantityStepper from "../components/QuantityStepper";
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
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [moveTargets, setMoveTargets] = useState<Collection[]>([]);

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

  const userId = token ? parseUserId(token) : null;
  const isOwner = collection && userId !== null && String(userId) === collection.ownerId;

  // Rows can disappear (e.g. merged by a language change), so only count
  // selected ids that are still present.
  const allIds = ((items ?? []) as Array<{ id: string }>).map((i) => i.id);
  const selectedIds = allIds.filter((i) => selected.has(i));

  // Other collections of the owner's that can hold this collection's items.
  useEffect(() => {
    if (!collection || !isOwner) return;
    apiFetch("/v1/collections").then(async (r) => {
      if (!r.ok) return;
      const data = await r.json();
      const all: Collection[] = data.collections ?? [];
      setMoveTargets(all.filter((c) => c.id !== collection.id && c.type === collection.type));
    });
  }, [collection, isOwner]);

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
      setSelected((prev) => {
        const next = new Set(prev);
        next.delete(itemId);
        return next;
      });
    }
  }

  function handleToggleSelect(itemId: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(itemId)) next.delete(itemId);
      else next.add(itemId);
      return next;
    });
  }

  // Both throw on failure so the toolbar can report it.
  async function handleMoveSelected(targetId: string) {
    if (!collection) return;
    const resp = await apiFetch(`/v1/collections/${id}/move-items`, {
      method: "POST",
      body: JSON.stringify({ itemIds: selectedIds, targetCollectionId: targetId }),
    });
    if (!resp.ok) throw new Error("move failed");
    setSelected(new Set());
    await fetchItems(collection);
  }

  async function handleDeleteSelected() {
    if (!collection) return;
    const n = selectedIds.length;
    if (!confirm(`Remove ${n} item${n === 1 ? "" : "s"} from this collection? This cannot be undone.`)) return;
    const resp = await apiFetch(`/v1/collections/${id}/remove-items`, {
      method: "POST",
      body: JSON.stringify({ itemIds: selectedIds }),
    });
    if (!resp.ok) throw new Error("delete failed");
    setSelected(new Set());
    await fetchItems(collection);
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

  async function handleLanguageChange(itemId: string, language: string) {
    if (!collection || !items) return;
    const item = (items as CardItem[]).find((i) => i.id === itemId);
    if (!item) return;
    const resp = await apiFetch(`/v1/cards/${itemId}`, {
      method: "PATCH",
      body: JSON.stringify({ quantity: item.quantity, notes: item.notes, language }),
    });
    // Changing to a language the collection already holds for this printing
    // merges the rows, so refetch.
    if (resp.ok) await fetchItems(collection);
  }

  if (loading) return <AppLayout><p className="loading">Loading…</p></AppLayout>;
  if (error || !collection) return <AppLayout><p className="error-page">{error ?? "Collection not found."}</p></AppLayout>;

  return (
    <AppLayout>
      <div className="page">
        <Link to="/collections" className="back-link">← All collections</Link>

        <div className="detail-header">
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", gap: 16 }}>
            <h1>{collection.name}</h1>
            {isOwner && !collection.isDefault && (
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
            {collection.isDefault && <span className="badge" title="Your default card collection">Default</span>}
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
          <>
            {isOwner && (
              <SelectionToolbar
                selectedCount={selectedIds.length}
                moveTargets={moveTargets}
                onSelectAll={() => setSelected(new Set(allIds))}
                onInvert={() => setSelected(new Set(allIds.filter((i) => !selected.has(i))))}
                onClear={() => setSelected(new Set())}
                onMove={handleMoveSelected}
                onDelete={handleDeleteSelected}
              />
            )}
            <ItemList
              collection={collection}
              items={items}
              isOwner={!!isOwner}
              selected={selected}
              onToggleSelect={handleToggleSelect}
              onRemove={handleRemoveItem}
              onQuantityChange={handleQuantityChange}
              onLanguageChange={handleLanguageChange}
            />
          </>
        )}
      </div>
    </AppLayout>
  );
}

interface SelectionToolbarProps {
  selectedCount: number;
  moveTargets: Collection[];
  onSelectAll: () => void;
  onInvert: () => void;
  onClear: () => void;
  onMove: (targetId: string) => Promise<void>;
  onDelete: () => Promise<void>;
}

function SelectionToolbar({ selectedCount, moveTargets, onSelectAll, onInvert, onClear, onMove, onDelete }: SelectionToolbarProps) {
  const [target, setTarget] = useState("");
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState<string | null>(null);

  async function run(action: () => Promise<void>, failMsg: string) {
    setBusy(true);
    setFailed(null);
    try {
      await action();
    } catch {
      setFailed(failMsg);
    }
    setBusy(false);
  }

  const none = selectedCount === 0;
  return (
    <div className="selection-toolbar">
      <button className="btn-ghost" onClick={onSelectAll}>Select all</button>
      <button className="btn-ghost" onClick={onInvert}>Invert selection</button>
      <button className="btn-ghost" onClick={onClear} disabled={selectedCount === 0}>Clear selection</button>
      <span className="selection-count">{selectedCount} selected</span>
      {failed && <span className="selection-error">{failed}</span>}
      <div className="selection-actions">
        <select
          value={target}
          onChange={(e) => setTarget(e.target.value)}
          disabled={moveTargets.length === 0}
          aria-label="Collection to move selected items to"
        >
          <option value="">{moveTargets.length === 0 ? "No other compatible collections" : "Move to…"}</option>
          {moveTargets.map((c) => (
            <option key={c.id} value={c.id}>{c.name}</option>
          ))}
        </select>
        <button
          className="btn-sm"
          disabled={none || !target || busy}
          onClick={() => run(() => onMove(target), "Move failed.")}
        >
          Move
        </button>
        <button
          className="btn-danger"
          disabled={none || busy}
          onClick={() => run(onDelete, "Delete failed.")}
        >
          Delete
        </button>
      </div>
    </div>
  );
}

interface ItemListProps {
  collection: Collection;
  items: CardItem[] | SealedItem[] | BookDvdItem[];
  isOwner: boolean;
  selected: Set<string>;
  onToggleSelect: (id: string) => void;
  onRemove: (id: string) => void;
  onQuantityChange: (id: string, delta: 1 | -1) => void;
  onLanguageChange: (id: string, language: string) => void;
}

function ItemList({ collection, items, onLanguageChange, ...rowProps }: ItemListProps) {
  const type = collection.type;
  if (type === "COLLECTION_TYPE_MTG_CARDS") {
    return <CardItemList items={items as CardItem[]} onLanguageChange={onLanguageChange} {...rowProps} />;
  }
  if (type === "COLLECTION_TYPE_MTG_SEALED") {
    return <SealedItemList items={items as SealedItem[]} {...rowProps} />;
  }
  return <BookDvdItemList items={items as BookDvdItem[]} {...rowProps} />;
}

interface RowProps {
  isOwner: boolean;
  selected: Set<string>;
  onToggleSelect: (id: string) => void;
  onRemove: (id: string) => void;
  onQuantityChange: (id: string, delta: 1 | -1) => void;
}

function SelectCheckbox({ id, label, selected, onToggleSelect }: {
  id: string;
  label: string;
  selected: Set<string>;
  onToggleSelect: (id: string) => void;
}) {
  return (
    <input
      type="checkbox"
      className="item-select"
      checked={selected.has(id)}
      onChange={() => onToggleSelect(id)}
      aria-label={`Select ${label}`}
    />
  );
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

type CardRowProps = RowProps & { onLanguageChange: (id: string, language: string) => void };

function CardItemList({ items, ...rowProps }: { items: CardItem[] } & CardRowProps) {
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
      <CardRows items={items} {...rowProps} />
    </>
  );
}

function CardRows({ items, isOwner, selected, onToggleSelect, onRemove, onQuantityChange, onLanguageChange }: { items: CardItem[] } & CardRowProps) {
  return (
    <div className="item-list">
      {items.map((item) => {
        const unit = unitPrice(item);
        const subtotal = unit !== null ? unit * item.quantity : null;
        return (
          <div key={item.id} className={rowClass(selected, item.id)}>
            {isOwner && <SelectCheckbox id={item.id} label={item.cardName} selected={selected} onToggleSelect={onToggleSelect} />}
            {item.imageUri && (
              <img src={item.imageUri} alt={item.cardName} className="card-thumb" />
            )}
            <div className="item-info">
              <span className="item-name">
                {item.scryfallUri ? (
                  <a href={item.scryfallUri} target="_blank" rel="noopener noreferrer" title="View on Scryfall">
                    {item.cardName}
                  </a>
                ) : item.cardName}
              </span>
              <span className="item-meta">
                {item.setCode.toUpperCase()} #{item.collectorNumber} &middot; {FOIL_TREATMENT_LABELS[item.foilTreatment]}
                {!isOwner && item.language && item.language !== "en" && <> &middot; {languageLabel(item.language)}</>}
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
            {isOwner && (
              <select
                className="category-select"
                value={item.language || "en"}
                onChange={(e) => onLanguageChange(item.id, e.target.value)}
                aria-label={`Language of ${item.cardName}`}
              >
                {/* Keep an unlisted stored code selectable. */}
                {!LANGUAGE_OPTIONS.some((l) => l.code === (item.language || "en")) && (
                  <option value={item.language}>{languageLabel(item.language)}</option>
                )}
                {LANGUAGE_OPTIONS.map((l) => (
                  <option key={l.code} value={l.code}>{l.label}</option>
                ))}
              </select>
            )}
            <QuantityStepper id={item.id} quantity={item.quantity} editable={isOwner} onQuantityChange={onQuantityChange} />
            {isOwner && <RemoveButton onRemove={() => onRemove(item.id)} />}
          </div>
        );
      })}
    </div>
  );
}

function SealedItemList({ items, isOwner, selected, onToggleSelect, onRemove, onQuantityChange }: { items: SealedItem[] } & RowProps) {
  return (
    <div className="item-list">
      {items.map((item) => {
        const typeLabel = SEALED_ITEM_TYPE_LABELS[item.itemType] ?? item.itemType;
        const displayName = item.name || (item.setCode ? item.setCode.toUpperCase() : null);
        return (
          <div key={item.id} className={rowClass(selected, item.id)}>
            {isOwner && <SelectCheckbox id={item.id} label={displayName ?? typeLabel} selected={selected} onToggleSelect={onToggleSelect} />}
            <div className="item-info">
              <span className="item-name">{displayName ?? typeLabel}</span>
              <span className="item-meta">
                {typeLabel}
                {item.setCode && displayName !== item.setCode.toUpperCase() && ` · ${item.setCode.toUpperCase()}`}
              </span>
              {item.notes && <span className="item-notes">{item.notes}</span>}
            </div>
            <QuantityStepper id={item.id} quantity={item.quantity} editable={isOwner} onQuantityChange={onQuantityChange} />
            {isOwner && <RemoveButton onRemove={() => onRemove(item.id)} />}
          </div>
        );
      })}
    </div>
  );
}

function BookDvdItemList({ items, isOwner, selected, onToggleSelect, onRemove, onQuantityChange }: { items: BookDvdItem[] } & RowProps) {
  return (
    <div className="item-list">
      {items.map((item) => (
        <div key={item.id} className={rowClass(selected, item.id)}>
          {isOwner && <SelectCheckbox id={item.id} label={item.title} selected={selected} onToggleSelect={onToggleSelect} />}
          <div className="item-info">
            <span className="item-name">{item.title}</span>
            {item.upc && <span className="item-meta">UPC: {item.upc}</span>}
            {item.notes && <span className="item-notes">{item.notes}</span>}
          </div>
          <QuantityStepper id={item.id} quantity={item.quantity} editable={isOwner} onQuantityChange={onQuantityChange} />
          {isOwner && <RemoveButton onRemove={() => onRemove(item.id)} />}
        </div>
      ))}
    </div>
  );
}

function rowClass(selected: Set<string>, id: string): string {
  return selected.has(id) ? "item-row selected" : "item-row";
}

function itemsEndpoint(type: Collection["type"], id: string): string | null {
  switch (type) {
    case "COLLECTION_TYPE_MTG_CARDS":
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
  if (type === "COLLECTION_TYPE_MTG_CARDS") {
    const item = (items as CardItem[]).find((i) => i.id === itemId)!;
    return { quantity: newQty, notes: item.notes };
  }
  if (type === "COLLECTION_TYPE_MTG_SEALED") {
    const item = (items as SealedItem[]).find((i) => i.id === itemId)!;
    return { itemType: item.itemType, setCode: item.setCode, name: item.name, quantity: newQty, notes: item.notes };
  }
  const item = (items as BookDvdItem[]).find((i) => i.id === itemId)!;
  return { title: item.title, quantity: newQty, notes: item.notes };
}
