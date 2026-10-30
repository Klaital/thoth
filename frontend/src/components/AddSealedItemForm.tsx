import { useEffect, useRef, useState } from "react";
import { apiFetch } from "../api/auth";
import {
  SealedItemType,
  SEALED_ITEM_TYPE_LABELS,
  SEALED_ITEM_TYPE_OPTIONS,
} from "../api/types";

interface Props {
  collectionId: number;
  onSuccess: () => void;
  onItemAdded?: () => void;
}

export default function AddSealedItemForm({ collectionId, onSuccess, onItemAdded }: Props) {
  const [upc, setUpc] = useState("");
  const [upcLooking, setUpcLooking] = useState(false);
  const [upcError, setUpcError] = useState<string | null>(null);
  const [upcSetName, setUpcSetName] = useState("");
  const [filledViaUpc, setFilledViaUpc] = useState(false);

  const [itemType, setItemType] = useState<SealedItemType>("SEALED_ITEM_TYPE_PLAY_BOOSTER_BOX");
  const [setCode, setSetCode] = useState("");
  const [name, setName] = useState("");
  const [quantity, setQuantity] = useState(1);
  const [notes, setNotes] = useState("");
  const [setCodes, setSetCodes] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [addedCount, setAddedCount] = useState(0);

  const upcRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    apiFetch("/v1/scryfall/set-codes")
      .then(async (r) => {
        if (r.ok) {
          const data = await r.json();
          setSetCodes(data.setCodes ?? []);
        }
      })
      .catch(() => {});
  }, []);

  async function lookupUPC() {
    const code = upc.trim();
    if (!code) return;
    setUpcLooking(true);
    setUpcError(null);
    try {
      const r = await apiFetch(`/v1/upc/${encodeURIComponent(code)}`);
      if (!r.ok) {
        setUpcError(r.status === 404 ? "Barcode not found." : "Lookup failed — fill in manually.");
        return;
      }
      const data = await r.json();
      if (data.itemType && data.itemType !== "SEALED_ITEM_TYPE_UNSPECIFIED") {
        setItemType(data.itemType as SealedItemType);
      }
      if (data.setCode) setSetCode(data.setCode);
      setUpcSetName(data.setName ?? "");
      setFilledViaUpc(true);
    } finally {
      setUpcLooking(false);
    }
  }

  function handleUpcKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") {
      e.preventDefault();
      lookupUPC();
    }
  }

  const isSecretLair = itemType === "SEALED_ITEM_TYPE_SECRET_LAIR";

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      const body: Record<string, unknown> = {
        collectionId,
        itemType,
        quantity,
        notes,
      };
      if (setCode.trim()) body.setCode = setCode.trim().toLowerCase();
      if (name.trim()) body.name = name.trim();

      const resp = await apiFetch(`/v1/collections/${collectionId}/sealed`, {
        method: "POST",
        body: JSON.stringify(body),
      });
      if (!resp.ok) {
        setError("Failed to add sealed item.");
        return;
      }
      onItemAdded?.();
      setName("");
      setSetCode("");
      setQuantity(1);
      setNotes("");
      setAddedCount((c) => c + 1);

      if (filledViaUpc) {
        setUpc("");
        setUpcSetName("");
        setFilledViaUpc(false);
        // Yield to the DOM before re-focusing so the state flush completes.
        setTimeout(() => upcRef.current?.focus(), 0);
      }
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={handleSubmit}>
      <div className="field">
        <label htmlFor="upc">Barcode (UPC)</label>
        <div style={{ display: "flex", gap: "0.5rem", alignItems: "center" }}>
          <input
            id="upc"
            ref={upcRef}
            value={upc}
            onChange={(e) => { setUpc(e.target.value); setUpcError(null); }}
            onBlur={() => { if (upc.trim()) lookupUPC(); }}
            onKeyDown={handleUpcKeyDown}
            placeholder="Scan or type barcode…"
            autoComplete="off"
            inputMode="numeric"
            style={{ flex: 1 }}
          />
          {upcLooking && <span style={{ color: "var(--text-muted)", fontSize: "0.85rem" }}>Looking up…</span>}
        </div>
        {upcError && <p className="error-msg" role="alert" style={{ marginTop: "0.25rem" }}>{upcError}</p>}
      </div>

      <div className="field">
        <label htmlFor="item-type">Type</label>
        <select
          id="item-type"
          value={itemType}
          onChange={(e) => setItemType(e.target.value as SealedItemType)}
        >
          {SEALED_ITEM_TYPE_OPTIONS.map((t) => (
            <option key={t} value={t}>{SEALED_ITEM_TYPE_LABELS[t]}</option>
          ))}
        </select>
      </div>

      <div className="field">
        <label htmlFor="set-code">
          Set code{" "}
          <span style={{ color: "var(--text-muted)", fontWeight: 400 }}>(optional)</span>
        </label>
        <input
          id="set-code"
          list="set-codes-list"
          value={setCode}
          onChange={(e) => setSetCode(e.target.value)}
          placeholder="e.g. mh3"
          maxLength={16}
          autoComplete="off"
        />
        {setCodes.length > 0 && (
          <datalist id="set-codes-list">
            {setCodes.map((sc) => <option key={sc} value={sc} />)}
          </datalist>
        )}
        {upcSetName && (
          <p style={{ margin: "0.25rem 0 0", fontSize: "0.82rem", color: "var(--text-muted)" }}>
            Set identified as: <em>{upcSetName}</em>
          </p>
        )}
      </div>

      {isSecretLair && (
        <div className="field">
          <label htmlFor="name">Secret Lair name</label>
          <input
            id="name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. Secret Lair: Through the Wormhole"
          />
        </div>
      )}

      {!isSecretLair && (
        <div className="field">
          <label htmlFor="name">
            Name{" "}
            <span style={{ color: "var(--text-muted)", fontWeight: 400 }}>(optional)</span>
          </label>
          <input
            id="name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Optional label"
          />
        </div>
      )}

      <div className="field">
        <label htmlFor="quantity">Quantity</label>
        <input
          id="quantity"
          type="number"
          min={1}
          value={quantity}
          onChange={(e) => setQuantity(Number(e.target.value))}
        />
      </div>

      <div className="field">
        <label htmlFor="notes">
          Notes <span style={{ color: "var(--text-muted)", fontWeight: 400 }}>(optional)</span>
        </label>
        <textarea
          id="notes"
          rows={2}
          value={notes}
          onChange={(e) => setNotes(e.target.value)}
          style={{ resize: "vertical" }}
        />
      </div>

      {error && <p className="error-msg" role="alert">{error}</p>}

      <div className="form-actions">
        <button type="submit" className="btn-primary" disabled={submitting}>
          {submitting ? "Adding…" : "Add to collection"}
        </button>
        <button type="button" className="btn-secondary" onClick={onSuccess}>
          Done
        </button>
        {addedCount > 0 && (
          <span className="added-confirmation">
            {addedCount} added
          </span>
        )}
      </div>
    </form>
  );
}
