import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { apiFetch } from "../api/auth";
import { describePrinting, groupCheckoutsByCollection } from "../api/checkout";
import { errorMessage } from "../api/errors";
import { Deck, DeckCheckout } from "../api/types";
import AppLayout from "../components/AppLayout";

// Return wizard (TH-17): the deck's checked-out cards grouped by the box they
// came from. Tick them off while putting them back, then confirm.
export default function DeckReturnPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [deck, setDeck] = useState<Deck | null>(null);
  const [checkouts, setCheckouts] = useState<DeckCheckout[] | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [returned, setReturned] = useState<Set<string>>(new Set());
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([apiFetch(`/v1/decks/${id}`), apiFetch(`/v1/decks/${id}/checkouts`)])
      .then(async ([d, c]) => {
        if (!d.ok || !c.ok) { setLoadError(await errorMessage(d.ok ? c : d, "Deck not found.")); return; }
        setDeck((await d.json()).deck);
        setCheckouts((await c.json()).checkouts ?? []);
      })
      .catch(() => setLoadError("Failed to load the deck."));
  }, [id]);

  if (loadError) return <AppLayout><p className="error-page">{loadError}</p></AppLayout>;
  if (!deck || !checkouts) return <AppLayout><p className="loading">Loading…</p></AppLayout>;

  const groups = groupCheckoutsByCollection(checkouts);
  const total = checkouts.reduce((n, c) => n + c.quantity, 0);
  const ticked = checkouts.filter((c) => returned.has(c.collectionCardId));
  const tickedTotal = ticked.reduce((n, c) => n + c.quantity, 0);

  function toggle(copyId: string) {
    setReturned((prev) => {
      const next = new Set(prev);
      if (next.has(copyId)) next.delete(copyId);
      else next.add(copyId);
      return next;
    });
  }

  async function handleConfirm() {
    setSubmitting(true);
    setSubmitError(null);
    try {
      const resp = await apiFetch(`/v1/decks/${id}/return`, {
        method: "POST",
        body: JSON.stringify({
          picks: ticked.map((c) => ({ collectionCardId: c.collectionCardId, quantity: c.quantity })),
        }),
      });
      if (!resp.ok) { setSubmitError(await errorMessage(resp, "Return failed.")); return; }
      navigate(`/decks/${id}`);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <AppLayout>
      <div className="page">
        <Link to={`/decks/${id}`} className="back-link">← {deck.name}</Link>
        <div className="page-header">
          <h1>Return {deck.name}</h1>
          {checkouts.length > 0 && (
            <button
              className="btn-ghost"
              onClick={() => setReturned(tickedTotal === total ? new Set() : new Set(checkouts.map((c) => c.collectionCardId)))}
            >
              {tickedTotal === total ? "Untick all" : "Tick all"}
            </button>
          )}
        </div>

        {checkouts.length === 0 ? (
          <div className="empty-state" style={{ padding: "32px 0" }}>
            <p>No cards are checked out to this deck.</p>
          </div>
        ) : (
          <>
            <p className="item-meta" role="status">
              {tickedTotal} of {total} put back. Tick each card as it goes back in its box.
            </p>
            {groups.map((g) => (
              <section key={g.collectionId} className="card-category-group">
                <h3><Link to={`/collections/${g.collectionId}`}>{g.collectionName}</Link></h3>
                <div className="item-list">
                  {g.lines.map((c) => (
                    <label key={c.collectionCardId} className={`item-row checklist-row${returned.has(c.collectionCardId) ? " selected" : ""}`}>
                      <input type="checkbox" checked={returned.has(c.collectionCardId)} onChange={() => toggle(c.collectionCardId)} />
                      {c.imageUri && <img src={c.imageUri} alt="" className="card-thumb" />}
                      <span className="item-info">
                        <span className="item-name">{c.quantity}× {c.cardName}</span>
                        <span className="item-meta">{describePrinting(c)}</span>
                      </span>
                    </label>
                  ))}
                </div>
              </section>
            ))}
            {submitError && <p className="error-msg" role="alert">{submitError}</p>}
            <div className="wizard-actions">
              <button className="btn-sm" disabled={tickedTotal === 0 || submitting} onClick={handleConfirm}>
                {submitting ? "Saving…" : `Return ${tickedTotal} ${tickedTotal === 1 ? "card" : "cards"}`}
              </button>
            </div>
          </>
        )}
      </div>
    </AppLayout>
  );
}
