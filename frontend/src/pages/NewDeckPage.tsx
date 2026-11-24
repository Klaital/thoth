import { useState } from "react";
import { useNavigate, Link } from "react-router-dom";
import { apiFetch } from "../api/auth";
import AppLayout from "../components/AppLayout";

export default function NewDeckPage() {
  const navigate = useNavigate();
  const [name, setName] = useState("");
  const [notes, setNotes] = useState("");
  const [isPublic, setIsPublic] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const resp = await apiFetch("/v1/decks", {
        method: "POST",
        body: JSON.stringify({ name, notes, isPublic }),
      });
      if (!resp.ok) {
        setError("Failed to create deck.");
        return;
      }
      const data = await resp.json();
      navigate(`/decks/${data.deck.id}`);
    } catch {
      setError("Could not reach the server. Please try again.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AppLayout>
      <div className="page">
        <Link to="/decks" className="back-link">← Back to decks</Link>
        <div className="form-card">
          <h1>New Deck</h1>
          <form onSubmit={handleSubmit}>
            <div className="field">
              <label htmlFor="name">Name</label>
              <input
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. Atraxa Superfriends"
                required
              />
            </div>

            <div className="field">
              <label htmlFor="notes">Notes <span style={{ color: "var(--text-muted)", fontWeight: 400 }}>(optional)</span></label>
              <textarea
                id="notes"
                rows={4}
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                placeholder="Where it's stored, how it plays, ideas to improve it…"
                style={{ resize: "vertical" }}
              />
            </div>

            <div className="checkbox-row">
              <input
                id="isPublic"
                type="checkbox"
                checked={isPublic}
                onChange={(e) => setIsPublic(e.target.checked)}
              />
              <label htmlFor="isPublic">Make this deck public</label>
            </div>

            {error && <p className="error-msg" role="alert">{error}</p>}

            <button type="submit" className="btn-primary" disabled={loading || !name.trim()}>
              {loading ? "Creating…" : "Create deck"}
            </button>
          </form>
        </div>
      </div>
    </AppLayout>
  );
}
