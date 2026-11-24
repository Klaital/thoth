import { useCallback, useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { apiFetch, parseUserId } from "../api/auth";
import { Deck, DeckBoard, DeckCard, DECK_BOARD_LABELS, DECK_BOARD_OPTIONS } from "../api/types";
import { describeDeckLine } from "../api/deckText";
import AppLayout from "../components/AppLayout";
import BulkImportForm from "../components/BulkImportForm";
import CommanderSummary from "../components/CommanderSummary";
import QuantityStepper from "../components/QuantityStepper";
import { useAuth } from "../context/AuthContext";

const DECK_BULK_PLACEHOLDER = `Commander
1 Atraxa, Praetors' Voice
Deck
1 Sol Ring
4 Lightning Bolt (M20) 141
Sideboard
2 Swords to Plowshares`;

// Error message from a gRPC-gateway error response, if any.
async function errorMessage(resp: Response, fallback: string): Promise<string> {
  try {
    const data = await resp.json();
    return data.message || fallback;
  } catch {
    return fallback;
  }
}

export default function DeckDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { token } = useAuth();
  const navigate = useNavigate();
  const [deck, setDeck] = useState<Deck | null>(null);
  const [cards, setCards] = useState<DeckCard[]>([]);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [bulkOpen, setBulkOpen] = useState(false);

  const fetchDeck = useCallback(async () => {
    const r = await apiFetch(`/v1/decks/${id}`);
    if (!r.ok) { setNotFound(true); return; }
    const data = await r.json();
    setDeck(data.deck ?? null);
  }, [id]);

  const fetchCards = useCallback(async () => {
    const r = await apiFetch(`/v1/decks/${id}/cards`);
    if (!r.ok) return;
    const data = await r.json();
    setCards(data.cards ?? []);
  }, [id]);

  // Cards and commanders change together, so refresh both.
  const refresh = useCallback(async () => {
    await Promise.all([fetchDeck(), fetchCards()]);
  }, [fetchDeck, fetchCards]);

  useEffect(() => {
    refresh().finally(() => setLoading(false));
  }, [refresh]);

  const userId = token ? parseUserId(token) : null;
  const isOwner = !!deck && userId !== null && String(userId) === deck.ownerId;

  async function handleDelete() {
    if (!confirm("Delete this deck? This cannot be undone.")) return;
    const resp = await apiFetch(`/v1/decks/${id}`, { method: "DELETE" });
    if (resp.ok) navigate("/decks");
  }

  async function handleQuantityChange(cardId: string, delta: 1 | -1) {
    const current = cards.find((c) => c.id === cardId);
    if (!current) return;
    const quantity = Math.max(1, current.quantity + delta);
    if (quantity === current.quantity) return;
    setCards((prev) => prev.map((c) => (c.id === cardId ? { ...c, quantity } : c)));
    const resp = await apiFetch(`/v1/deck-cards/${cardId}`, { method: "PATCH", body: JSON.stringify({ quantity }) });
    if (!resp.ok) {
      setCards((prev) => prev.map((c) => (c.id === cardId ? { ...c, quantity: current.quantity } : c)));
    }
  }

  async function handleBoardChange(cardId: string, board: DeckBoard) {
    const resp = await apiFetch(`/v1/deck-cards/${cardId}`, { method: "PATCH", body: JSON.stringify({ board }) });
    // Moving onto a board that already has this card merges the rows.
    if (resp.ok) await refresh();
  }

  async function handleRemove(cardId: string) {
    const resp = await apiFetch(`/v1/deck-cards/${cardId}`, { method: "DELETE" });
    if (resp.ok) await refresh();
  }

  if (loading) return <AppLayout><p className="loading">Loading…</p></AppLayout>;
  if (notFound || !deck) return <AppLayout><p className="error-page">Deck not found.</p></AppLayout>;

  const total = cards.reduce((n, c) => n + c.quantity, 0);

  return (
    <AppLayout>
      <div className="page">
        <Link to="/decks" className="back-link">← All decks</Link>

        <DeckHeader deck={deck} isOwner={isOwner} onSaved={setDeck} onDelete={handleDelete} />

        <div className="page-header" style={{ marginBottom: 16 }}>
          <h2 style={{ margin: 0, fontSize: 16, fontWeight: 600 }}>Cards ({total})</h2>
          {isOwner && (
            <button className="btn-sm" onClick={() => setBulkOpen((o) => !o)}>
              {bulkOpen ? "Close bulk import" : "Bulk import"}
            </button>
          )}
        </div>

        {isOwner && bulkOpen && (
          <div className="form-card" style={{ marginBottom: 24 }}>
            <BulkImportForm
              url={`/v1/decks/${id}/cards/bulk`}
              placeholder={DECK_BULK_PLACEHOLDER}
              describeLine={describeDeckLine}
              help={
                <>
                  One card per line: <em>&lt;qty&gt; &lt;name&gt;</em>, or Moxfield format{" "}
                  <em>&lt;qty&gt; &lt;name&gt; (&lt;set&gt;) &lt;number&gt;</em> to also record a preferred printing.
                  Header lines <code>Commander</code>, <code>Deck</code> and <code>Sideboard</code> set the board
                  of the cards below them; cards before the first header go in Main.
                </>
              }
              onItemAdded={refresh}
              onSuccess={() => setBulkOpen(false)}
            />
          </div>
        )}

        {isOwner && !bulkOpen && <AddDeckCardForm deckId={id!} onAdded={refresh} />}

        {cards.length === 0 ? (
          <div className="empty-state" style={{ padding: "32px 0" }}>
            <p>No cards yet.{isOwner ? " Add some by name, or bulk import a list." : ""}</p>
          </div>
        ) : (
          DECK_BOARD_OPTIONS.map((board) => {
            const group = cards.filter((c) => c.board === board);
            if (group.length === 0) return null;
            const count = group.reduce((n, c) => n + c.quantity, 0);
            return (
              <section key={board} className="card-category-group">
                <h3>{DECK_BOARD_LABELS[board]} ({count})</h3>
                <div className="item-list">
                  {group.map((card) => (
                    <DeckCardRow
                      key={card.id}
                      card={card}
                      isOwner={isOwner}
                      onQuantityChange={handleQuantityChange}
                      onBoardChange={handleBoardChange}
                      onRemove={handleRemove}
                    />
                  ))}
                </div>
              </section>
            );
          })
        )}
      </div>
    </AppLayout>
  );
}

function DeckHeader({ deck, isOwner, onSaved, onDelete }: {
  deck: Deck;
  isOwner: boolean;
  onSaved: (deck: Deck) => void;
  onDelete: () => void;
}) {
  const [name, setName] = useState(deck.name);
  const [notes, setNotes] = useState(deck.notes ?? "");
  const [isPublic, setIsPublic] = useState(deck.isPublic);
  const [saving, setSaving] = useState(false);
  const [status, setStatus] = useState<string | null>(null);

  // Keep the form in sync when the deck is refetched (e.g. after a card change).
  useEffect(() => {
    setName(deck.name);
    setNotes(deck.notes ?? "");
    setIsPublic(deck.isPublic);
  }, [deck.name, deck.notes, deck.isPublic]);

  const dirty = name !== deck.name || notes !== (deck.notes ?? "") || isPublic !== deck.isPublic;

  async function handleSave() {
    if (!name.trim()) { setStatus("Name is required."); return; }
    setSaving(true);
    setStatus(null);
    const resp = await apiFetch(`/v1/decks/${deck.id}`, {
      method: "PATCH",
      body: JSON.stringify({ name, notes, isPublic }),
    });
    if (resp.ok) {
      const data = await resp.json();
      onSaved(data.deck);
      setStatus("Saved.");
    } else {
      setStatus(await errorMessage(resp, "Save failed."));
    }
    setSaving(false);
  }

  return (
    <div className="detail-header">
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", gap: 16 }}>
        {isOwner ? (
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            aria-label="Deck name"
            style={{ fontSize: 24, fontWeight: 600, flex: 1 }}
          />
        ) : (
          <h1>{deck.name}</h1>
        )}
        {isOwner && <button className="btn-danger" onClick={onDelete}>Delete</button>}
      </div>
      {deck.commanders && deck.commanders.length > 0 && (
        <CommanderSummary commanders={deck.commanders} size="large" />
      )}
      {isOwner ? (
        <div className="field deck-notes" style={{ marginTop: 12 }}>
          <label htmlFor="deck-notes">Notes</label>
          <textarea
            id="deck-notes"
            rows={3}
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            placeholder="Where it's stored, how it plays, ideas to improve it…"
          />
          <div style={{ display: "flex", alignItems: "center", gap: 12, marginTop: 8 }}>
            <label className="checkbox-row" style={{ margin: 0 }}>
              <input type="checkbox" checked={isPublic} onChange={(e) => setIsPublic(e.target.checked)} />
              Public
            </label>
            <button className="btn-sm" onClick={handleSave} disabled={!dirty || saving}>
              {saving ? "Saving…" : "Save"}
            </button>
            {status && <span className="deck-notes-status" role="status">{status}</span>}
          </div>
        </div>
      ) : (
        <>
          {deck.notes && <p className="desc" style={{ whiteSpace: "pre-wrap" }}>{deck.notes}</p>}
          <div className="meta">
            {deck.isPublic && <span className="badge badge-public">Public</span>}
          </div>
        </>
      )}
    </div>
  );
}

interface NameSuggestion {
  name: string;
}

// Adds a card by name. Suggestions come from the Scryfall card search, which
// returns printings, so names are de-duplicated.
function AddDeckCardForm({ deckId, onAdded }: { deckId: string; onAdded: () => Promise<void> }) {
  const [name, setName] = useState("");
  const [board, setBoard] = useState<DeckBoard>("DECK_BOARD_MAIN");
  const [quantity, setQuantity] = useState(1);
  const [suggestions, setSuggestions] = useState<NameSuggestion[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [lastAdded, setLastAdded] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    const q = name.trim();
    if (q.length < 2) { setSuggestions([]); return; }
    const controller = new AbortController();
    const timer = setTimeout(() => {
      apiFetch(`/v1/scryfall/search?query=${encodeURIComponent(q)}`, { signal: controller.signal })
        .then((r) => r.json())
        .then((data: { cards?: { name: string }[] }) => {
          const names = [...new Set((data.cards ?? []).map((c) => c.name))];
          // Hide the list once the input exactly matches the only suggestion.
          setSuggestions(names.length === 1 && names[0].toLowerCase() === q.toLowerCase() ? [] : names.map((n) => ({ name: n })));
        })
        .catch(() => {});
    }, 200);
    return () => { clearTimeout(timer); controller.abort(); };
  }, [name]);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    setError(null);
    setSubmitting(true);
    try {
      const resp = await apiFetch(`/v1/decks/${deckId}/cards`, {
        method: "POST",
        body: JSON.stringify({ cardName: name.trim(), board, quantity }),
      });
      if (!resp.ok) {
        setError(await errorMessage(resp, "Failed to add card."));
        return;
      }
      const data = await resp.json();
      setLastAdded(data.card?.cardName ?? name);
      setName("");
      setQuantity(1);
      setSuggestions([]);
      await onAdded();
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={handleSubmit}>
      <div className="deck-add-card">
        <div className="field name-field">
          <label htmlFor="deck-card-name">Add card</label>
          <input
            id="deck-card-name"
            value={name}
            onChange={(e) => { setName(e.target.value); setLastAdded(null); }}
            placeholder="Card name, e.g. City of Brass"
            autoComplete="off"
          />
          {suggestions.length > 0 && (
            <ul className="name-suggestions" role="listbox">
              {suggestions.map((s) => (
                <li key={s.name}>
                  <button type="button" onClick={() => { setName(s.name); setSuggestions([]); }}>{s.name}</button>
                </li>
              ))}
            </ul>
          )}
        </div>
        <div className="field">
          <label htmlFor="deck-card-board">Board</label>
          <select id="deck-card-board" value={board} onChange={(e) => setBoard(e.target.value as DeckBoard)}>
            {DECK_BOARD_OPTIONS.map((b) => <option key={b} value={b}>{DECK_BOARD_LABELS[b]}</option>)}
          </select>
        </div>
        <div className="field" style={{ width: 72 }}>
          <label htmlFor="deck-card-qty">Qty</label>
          <input id="deck-card-qty" type="number" min={1} value={quantity} onChange={(e) => setQuantity(Number(e.target.value))} />
        </div>
        <button type="submit" className="btn-sm" disabled={submitting || !name.trim()}>
          {submitting ? "Adding…" : "Add"}
        </button>
      </div>
      {error && <p className="error-msg" role="alert">{error}</p>}
      {lastAdded && <div className="success-banner" role="status">✓ {lastAdded} added.</div>}
    </form>
  );
}

function DeckCardRow({ card, isOwner, onQuantityChange, onBoardChange, onRemove }: {
  card: DeckCard;
  isOwner: boolean;
  onQuantityChange: (id: string, delta: 1 | -1) => void;
  onBoardChange: (id: string, board: DeckBoard) => void;
  onRemove: (id: string) => void;
}) {
  return (
    <div className="item-row">
      {card.imageUri && <img src={card.imageUri} alt={card.cardName} className="card-thumb" />}
      <div className="item-info">
        <span className="item-name">
          {card.scryfallUri ? (
            <a href={card.scryfallUri} target="_blank" rel="noopener noreferrer" title="View on Scryfall">
              {card.cardName}
            </a>
          ) : card.cardName}
        </span>
        {card.preferredSetCode && (
          <span className="item-meta">
            Preferred: {card.preferredSetCode.toUpperCase()} #{card.preferredCollectorNumber}
          </span>
        )}
      </div>
      {isOwner && (
        <select
          className="category-select"
          value={card.board}
          onChange={(e) => onBoardChange(card.id, e.target.value as DeckBoard)}
          aria-label={`Board for ${card.cardName}`}
        >
          {DECK_BOARD_OPTIONS.map((b) => <option key={b} value={b}>{DECK_BOARD_LABELS[b]}</option>)}
        </select>
      )}
      <QuantityStepper id={card.id} quantity={card.quantity} editable={isOwner} onQuantityChange={onQuantityChange} />
      {isOwner && (
        <button
          className="btn-ghost"
          onClick={() => onRemove(card.id)}
          title="Remove from deck"
          style={{ color: "var(--danger)", flexShrink: 0, padding: "4px 8px", fontSize: 13 }}
        >
          ✕
        </button>
      )}
    </div>
  );
}
