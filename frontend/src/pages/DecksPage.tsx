import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { apiFetch } from "../api/auth";
import { Deck } from "../api/types";
import AppLayout from "../components/AppLayout";
import CommanderSummary from "../components/CommanderSummary";

export default function DecksPage() {
  const [decks, setDecks] = useState<Deck[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    apiFetch("/v1/decks")
      .then(async (r) => {
        if (!r.ok) { setError("Failed to load decks."); return; }
        const data = await r.json();
        setDecks(data.decks ?? []);
      })
      .catch(() => setError("Failed to load decks."))
      .finally(() => setLoading(false));
  }, []);

  return (
    <AppLayout>
      <div className="page">
        <div className="page-header">
          <h1>My Decks</h1>
          <Link to="/decks/new">
            <button className="btn-sm">+ New Deck</button>
          </Link>
        </div>

        {loading ? (
          <p className="loading">Loading…</p>
        ) : error ? (
          <p className="error-msg" role="alert">{error}</p>
        ) : decks.length === 0 ? (
          <div className="empty-state">
            <h2>No decks yet</h2>
            <p>Decks list cards by name, so you can plan one before you own the cards.</p>
            <Link to="/decks/new">
              <button className="btn-sm">Create a deck</button>
            </Link>
          </div>
        ) : (
          <div className="collection-grid">
            {decks.map((d) => (
              <Link key={d.id} to={`/decks/${d.id}`} className="collection-card">
                <h2>{d.name}</h2>
                {d.commanders && d.commanders.length > 0 && <CommanderSummary commanders={d.commanders} />}
                {d.notes && <p className="card-desc">{d.notes}</p>}
                <div className="card-meta">
                  <span className="badge badge-mtg">Deck</span>
                  {d.isPublic && <span className="badge badge-public">Public</span>}
                </div>
              </Link>
            ))}
          </div>
        )}
      </div>
    </AppLayout>
  );
}
