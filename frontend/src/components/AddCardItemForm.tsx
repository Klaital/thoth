import { useState } from "react";
import { apiFetch } from "../api/auth";

interface Props {
  collectionId: number;
  onDone: () => void;
  onItemAdded?: () => void;
}

interface ScryfallCard {
  id: string;
  name: string;
  setCode: string;
  collectorNumber: string;
  imageUri: string;
}

const FOIL_OPTIONS = [
  { value: "FOIL_TREATMENT_NONFOIL", label: "Non-foil" },
  { value: "FOIL_TREATMENT_FOIL",    label: "Foil" },
  { value: "FOIL_TREATMENT_ETCHED",  label: "Etched" },
];

export default function AddCardItemForm({ collectionId, onDone, onItemAdded }: Props) {
  const [query, setQuery] = useState("");
  const [setCode, setSetCode] = useState("");
  const [results, setResults] = useState<ScryfallCard[]>([]);
  const [selected, setSelected] = useState<ScryfallCard | null>(null);
  const [foil, setFoil] = useState("FOIL_TREATMENT_NONFOIL");
  const [quantity, setQuantity] = useState(1);
  const [notes, setNotes] = useState("");
  const [searching, setSearching] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [lastAdded, setLastAdded] = useState<string | null>(null);

  async function handleSearch() {
    if (query.trim().length < 2) return;
    setLastAdded(null);
    setSearching(true);
    setResults([]);
    try {
      const params = new URLSearchParams({ query });
      if (setCode.trim()) params.set("setCode", setCode.trim().toLowerCase());
      const resp = await apiFetch(`/v1/scryfall/search?${params}`);
      const data = await resp.json();
      setResults(data.cards ?? []);
    } finally {
      setSearching(false);
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!selected) return;
    setError(null);
    setSubmitting(true);
    try {
      const resp = await apiFetch(`/v1/collections/${collectionId}/cards`, {
        method: "POST",
        body: JSON.stringify({
          collectionId,
          scryfallCardId: selected.id,
          foilTreatment: foil,
          quantity,
          notes,
        }),
      });
      if (!resp.ok) {
        setError("Failed to add card.");
        return;
      }
      // Stay on the page; reset to search but keep set code and show confirmation.
      const addedName = selected.name;
      setSelected(null);
      setResults([]);
      setQuery("");
      setFoil("FOIL_TREATMENT_NONFOIL");
      setQuantity(1);
      setNotes("");
      setLastAdded(addedName);
      onItemAdded?.();
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={handleSubmit}>
      {lastAdded && (
        <div className="success-banner" role="status">
          ✓ {lastAdded} added.
        </div>
      )}

      {!selected ? (
        <>
          <div style={{ display: "flex", gap: 8, marginBottom: 16 }}>
            <div className="field" style={{ flex: "0 0 80px", marginBottom: 0 }}>
              <label htmlFor="set-filter">Set</label>
              <input
                id="set-filter"
                value={setCode}
                onChange={(e) => setSetCode(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && (e.preventDefault(), handleSearch())}
                placeholder="e.g. mh3"
                maxLength={8}
                style={{ textTransform: "lowercase" }}
              />
            </div>
            <div className="field" style={{ flex: 1, marginBottom: 0 }}>
              <label htmlFor="card-search">Card name</label>
              <div style={{ display: "flex", gap: 8 }}>
                <input
                  id="card-search"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && (e.preventDefault(), handleSearch())}
                  placeholder="Name…"
                />
                <button type="button" className="btn-sm" onClick={handleSearch} disabled={searching} style={{ flexShrink: 0 }}>
                  {searching ? "…" : "Search"}
                </button>
              </div>
            </div>
          </div>

          {results.length > 0 && (
            <div style={{ display: "flex", flexDirection: "column", gap: 8, marginBottom: 16 }}>
              {results.map((card) => (
                <button
                  key={card.id}
                  type="button"
                  className="type-option"
                  onClick={() => { setSelected(card); setLastAdded(null); }}
                  style={{ display: "flex", alignItems: "center", gap: 12 }}
                >
                  {card.imageUri && (
                    <img src={card.imageUri} alt={card.name} style={{ width: 36, borderRadius: 4 }} />
                  )}
                  <span>
                    <strong>{card.name}</strong>
                    <span style={{ color: "var(--text-muted)", marginLeft: 8 }}>
                      {card.setCode.toUpperCase()} #{card.collectorNumber}
                    </span>
                  </span>
                </button>
              ))}
            </div>
          )}
          {results.length === 0 && query && !searching && (
            <p style={{ color: "var(--text-muted)", fontSize: 13 }}>No results. Try running a Scryfall sync first.</p>
          )}

          <button type="button" className="btn-ghost" onClick={onDone} style={{ marginTop: 8 }}>
            ← Done adding cards
          </button>
        </>
      ) : (
        <>
          <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 20, padding: "12px", background: "var(--bg)", borderRadius: "var(--radius)", border: "1px solid var(--border)" }}>
            {selected.imageUri && (
              <img src={selected.imageUri} alt={selected.name} style={{ width: 44, borderRadius: 4 }} />
            )}
            <div style={{ flex: 1 }}>
              <div style={{ fontWeight: 600 }}>{selected.name}</div>
              <div style={{ color: "var(--text-muted)", fontSize: 13 }}>{selected.setCode.toUpperCase()} #{selected.collectorNumber}</div>
            </div>
            <button type="button" className="btn-ghost" onClick={() => setSelected(null)} style={{ fontSize: 12 }}>Change</button>
          </div>

          <div className="field">
            <label>Foil treatment</label>
            <div style={{ display: "flex", gap: 8 }}>
              {FOIL_OPTIONS.map((o) => (
                <button
                  key={o.value}
                  type="button"
                  className={`type-option${foil === o.value ? " selected" : ""}`}
                  onClick={() => setFoil(o.value)}
                  style={{ flex: 1 }}
                >
                  {o.label}
                </button>
              ))}
            </div>
          </div>

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
            <label htmlFor="notes">Notes <span style={{ color: "var(--text-muted)", fontWeight: 400 }}>(optional)</span></label>
            <textarea id="notes" rows={2} value={notes} onChange={(e) => setNotes(e.target.value)} style={{ resize: "vertical" }} />
          </div>

          {error && <p className="error-msg" role="alert">{error}</p>}

          <button type="submit" className="btn-primary" disabled={submitting}>
            {submitting ? "Adding…" : "Add to collection"}
          </button>
        </>
      )}
    </form>
  );
}
