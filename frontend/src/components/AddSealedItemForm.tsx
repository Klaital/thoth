import { useState } from "react";
import { apiFetch } from "../api/auth";

interface Props {
  collectionId: number;
  onSuccess: () => void;
}

export default function AddSealedItemForm({ collectionId, onSuccess }: Props) {
  const [setCode, setSetCode] = useState("");
  const [description, setDescription] = useState("");
  const [quantity, setQuantity] = useState(1);
  const [notes, setNotes] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!setCode.trim()) {
      setError("Set code is required.");
      return;
    }
    setError(null);
    setSubmitting(true);
    try {
      const resp = await apiFetch(`/v1/collections/${collectionId}/sealed`, {
        method: "POST",
        body: JSON.stringify({
          collectionId,
          setCode: setCode.trim().toLowerCase(),
          description,
          quantity,
          notes,
        }),
      });
      if (!resp.ok) {
        setError("Failed to add sealed item.");
        return;
      }
      onSuccess();
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={handleSubmit}>
      <div className="field">
        <label htmlFor="set-code">Set code</label>
        <input
          id="set-code"
          value={setCode}
          onChange={(e) => setSetCode(e.target.value)}
          placeholder="e.g. mh3"
          maxLength={10}
        />
      </div>

      <div className="field">
        <label htmlFor="description">
          Description <span style={{ color: "var(--text-muted)", fontWeight: 400 }}>(optional)</span>
        </label>
        <input
          id="description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="e.g. Draft Booster Box"
        />
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

      <button type="submit" className="btn-primary" disabled={submitting}>
        {submitting ? "Adding…" : "Add to collection"}
      </button>
    </form>
  );
}
