import { useEffect, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { apiFetch } from "../api/auth";
import {
  PickQuantities,
  describePrinting,
  initialPicks,
  pickedFor,
  pullList,
  stillNeeded,
  validatePicks,
} from "../api/checkout";
import { errorMessage } from "../api/errors";
import { Deck, DeckCardOwnership, DeckOwnership } from "../api/types";
import AppLayout from "../components/AppLayout";

// Check-out wizard (TH-17): choose which owned copies to pull, tick them off
// box by box while pulling, then record the ones that were pulled. Nothing
// is saved until the last step.
export default function DeckCheckoutPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  // ?card=<name>: a shortcut from one deck row, so only that card is shown.
  const onlyCard = useSearchParams()[0].get("card");
  const [deck, setDeck] = useState<Deck | null>(null);
  const [ownership, setOwnership] = useState<DeckOwnership | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [picks, setPicks] = useState<PickQuantities>({});
  const [step, setStep] = useState<"choose" | "pull">("choose");
  const [pulled, setPulled] = useState<Set<string>>(new Set());
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([apiFetch(`/v1/decks/${id}`), apiFetch(`/v1/decks/${id}/ownership`)])
      .then(async ([d, o]) => {
        if (!d.ok || !o.ok) { setLoadError(await errorMessage(d.ok ? o : d, "Deck not found.")); return; }
        const all: DeckOwnership = await o.json();
        const own = onlyCard ? { ...all, cards: (all.cards ?? []).filter((c) => c.cardName === onlyCard) } : all;
        setDeck((await d.json()).deck);
        setOwnership(own);
        setPicks(initialPicks(own));
      })
      .catch(() => setLoadError("Failed to load the deck."));
  }, [id, onlyCard]);

  if (loadError) return <AppLayout><p className="error-page">{loadError}</p></AppLayout>;
  if (!deck || !ownership) return <AppLayout><p className="loading">Loading…</p></AppLayout>;

  const cards = ownership.cards ?? [];
  const toPull = cards.filter((c) => stillNeeded(c) > 0 && (c.free ?? 0) > 0);
  const unavailable = cards.filter((c) => stillNeeded(c) > 0 && (c.free ?? 0) === 0);
  const done = cards.filter((c) => (c.needed ?? 0) > 0 && stillNeeded(c) === 0);
  const problems = validatePicks(ownership, picks);
  const groups = pullList(ownership, picks);
  const lines = groups.flatMap((g) => g.lines);
  const pickedTotal = lines.reduce((n, l) => n + l.quantity, 0);
  const pulledLines = lines.filter((l) => pulled.has(l.copy.collectionCardId));
  const pulledTotal = pulledLines.reduce((n, l) => n + l.quantity, 0);

  function setPick(copyId: string, quantity: number) {
    setPicks((prev) => {
      const next = { ...prev };
      if (quantity > 0) next[copyId] = quantity;
      else delete next[copyId];
      return next;
    });
  }

  function togglePulled(copyId: string) {
    setPulled((prev) => {
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
      const resp = await apiFetch(`/v1/decks/${id}/checkout`, {
        method: "POST",
        body: JSON.stringify({
          picks: pulledLines.map((l) => ({ collectionCardId: l.copy.collectionCardId, quantity: l.quantity })),
        }),
      });
      if (!resp.ok) { setSubmitError(await errorMessage(resp, "Check-out failed.")); return; }
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
          <h1>Check out {onlyCard ?? deck.name}</h1>
          <span className="wizard-steps">
            <span className={step === "choose" ? "active" : ""}>1. Choose copies</span>
            <span className={step === "pull" ? "active" : ""}>2. Pull cards</span>
          </span>
        </div>

        {step === "choose" && (
          <>
            {onlyCard && (
              <p className="item-meta">
                For {deck.name}. <Link to={`/decks/${id}/checkout`}>Check out the whole deck instead</Link>
              </p>
            )}
            {toPull.length === 0 && (
              <div className="empty-state" style={{ padding: "32px 0" }}>
                <p>
                  {onlyCard
                    ? `Nothing to pull: no free copies of ${onlyCard}, or the deck already has every copy it needs.`
                    : "Nothing to pull: every card you own for this deck is already checked out to it."}
                </p>
              </div>
            )}
            {toPull.map((card) => (
              <ChooseCard key={card.cardName} card={card} picks={picks} onChange={setPick} />
            ))}

            {unavailable.length > 0 && (
              <section className="card-category-group">
                <h3>Missing, skipped ({unavailable.reduce((n, c) => n + stillNeeded(c), 0)})</h3>
                <p className="item-meta">
                  {unavailable.map((c) => `${stillNeeded(c)} ${c.cardName}`).join(", ")}
                  {unavailable.some((c) => (c.checkedOutElsewhere ?? 0) > 0) && " (some copies are checked out to other decks)"}
                </p>
              </section>
            )}
            {done.length > 0 && (
              <p className="item-meta">{done.length} {done.length === 1 ? "card is" : "cards are"} already checked out to this deck.</p>
            )}

            {problems.map((p) => <p key={p} className="error-msg" role="alert">{p}</p>)}
            <div className="wizard-actions">
              <button
                className="btn-sm"
                disabled={problems.length > 0 || pickedTotal === 0}
                onClick={() => { setPulled(new Set()); setStep("pull"); }}
              >
                Next: pull {pickedTotal} {pickedTotal === 1 ? "card" : "cards"}
              </button>
            </div>
          </>
        )}

        {step === "pull" && (
          <>
            <p className="item-meta" role="status">
              {pulledTotal} of {pickedTotal} pulled. Tick each card as you take it out of its box.
            </p>
            {groups.map((g) => (
              <section key={g.collectionId} className="card-category-group">
                <h3>{g.collectionName}</h3>
                <div className="item-list">
                  {g.lines.map((l) => (
                    <label key={l.copy.collectionCardId} className={`item-row checklist-row${pulled.has(l.copy.collectionCardId) ? " selected" : ""}`}>
                      <input
                        type="checkbox"
                        checked={pulled.has(l.copy.collectionCardId)}
                        onChange={() => togglePulled(l.copy.collectionCardId)}
                      />
                      {l.copy.imageUri && <img src={l.copy.imageUri} alt="" className="card-thumb" />}
                      <span className="item-info">
                        <span className="item-name">{l.quantity}× {l.cardName}</span>
                        <span className="item-meta">{describePrinting(l.copy)}</span>
                      </span>
                    </label>
                  ))}
                </div>
              </section>
            ))}
            {pulledTotal < pickedTotal && pulledTotal > 0 && (
              <p className="item-meta">Unticked cards stay in their boxes and are not checked out.</p>
            )}
            {submitError && <p className="error-msg" role="alert">{submitError}</p>}
            <div className="wizard-actions">
              <button className="btn-ghost" onClick={() => setStep("choose")}>Back</button>
              <button className="btn-sm" disabled={pulledTotal === 0 || submitting} onClick={handleConfirm}>
                {submitting ? "Saving…" : `Check out ${pulledTotal} ${pulledTotal === 1 ? "card" : "cards"}`}
              </button>
            </div>
          </>
        )}
      </div>
    </AppLayout>
  );
}

function ChooseCard({ card, picks, onChange }: {
  card: DeckCardOwnership;
  picks: PickQuantities;
  onChange: (copyId: string, quantity: number) => void;
}) {
  const need = stillNeeded(card);
  const picked = pickedFor(card, picks);
  const copies = (card.copies ?? []).filter((c) => (c.free ?? 0) > 0);
  return (
    <section className="card-category-group">
      <h3>
        {card.cardName}{" "}
        <span className={`ownership-badge ${picked >= need ? "ownership-ok" : "ownership-partial"}`}>
          {picked}/{need}
        </span>
      </h3>
      <div className="item-list">
        {copies.map((c) => (
          <div key={c.collectionCardId} className="item-row">
            {c.imageUri && <img src={c.imageUri} alt="" className="card-thumb" />}
            <span className="item-info">
              <span className="item-name">{c.collectionName}</span>
              <span className="item-meta">
                {describePrinting(c)} · {c.free ?? 0} free
                {c.unitPriceUsd && <> · ${c.unitPriceUsd}</>}
              </span>
            </span>
            <input
              type="number"
              min={0}
              max={c.free ?? 0}
              value={picks[c.collectionCardId] ?? 0}
              onChange={(e) => onChange(c.collectionCardId, Math.max(0, Math.floor(Number(e.target.value) || 0)))}
              aria-label={`Copies of ${card.cardName} from ${c.collectionName} (${describePrinting(c)})`}
              style={{ width: 64 }}
            />
          </div>
        ))}
      </div>
    </section>
  );
}
