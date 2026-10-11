import { useState } from "react";
import { useNavigate, Link } from "react-router-dom";
import { apiFetch } from "../api/auth";
import { CollectionType, COLLECTION_TYPE_LABELS } from "../api/types";
import AppLayout from "../components/AppLayout";

const TYPES = Object.entries(COLLECTION_TYPE_LABELS).filter(
  ([v]) => v !== "COLLECTION_TYPE_UNSPECIFIED"
) as [CollectionType, string][];

export default function NewCollectionPage() {
  const navigate = useNavigate();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [type, setType] = useState<CollectionType>("COLLECTION_TYPE_MTG_CARDS");
  const [isPublic, setIsPublic] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const resp = await apiFetch("/v1/collections", {
        method: "POST",
        body: JSON.stringify({ name, description, type, isPublic }),
      });
      if (!resp.ok) {
        setError("Failed to create collection.");
        return;
      }
      const data = await resp.json();
      navigate(`/collections/${data.collection.id}`);
    } catch {
      setError("Could not reach the server. Please try again.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AppLayout>
      <div className="page">
        <Link to="/collections" className="back-link">← Back to collections</Link>
        <div className="form-card">
          <h1>New Collection</h1>
          <form onSubmit={handleSubmit}>
            <div className="field">
              <label htmlFor="name">Name</label>
              <input
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. My Commander Deck"
                required
              />
            </div>

            <div className="field">
              <label htmlFor="description">Description <span style={{ color: "var(--text-muted)", fontWeight: 400 }}>(optional)</span></label>
              <textarea
                id="description"
                rows={3}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="What's in this collection?"
                style={{ resize: "vertical" }}
              />
            </div>

            <div className="field">
              <label>Type</label>
              <div className="type-grid">
                {TYPES.map(([value, label]) => (
                  <button
                    key={value}
                    type="button"
                    className={`type-option${type === value ? " selected" : ""}`}
                    onClick={() => setType(value)}
                  >
                    {label}
                  </button>
                ))}
              </div>
            </div>

            <div className="checkbox-row">
              <input
                id="isPublic"
                type="checkbox"
                checked={isPublic}
                onChange={(e) => setIsPublic(e.target.checked)}
              />
              <label htmlFor="isPublic">Make this collection public</label>
            </div>

            {error && <p className="error-msg" role="alert">{error}</p>}

            <button type="submit" className="btn-primary" disabled={loading}>
              {loading ? "Creating…" : "Create collection"}
            </button>
          </form>
        </div>
      </div>
    </AppLayout>
  );
}
