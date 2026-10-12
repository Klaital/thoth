import { useState } from "react";
import { apiFetch } from "../api/auth";

interface Props {
  collectionId: number;
  onSuccess: () => void;
}

export default function AddBookDvdItemForm({ collectionId, onSuccess }: Props) {
  const [title, setTitle] = useState("");
  const [upc, setUpc] = useState("");
  const [quantity, setQuantity] = useState(1);
  const [notes, setNotes] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!title.trim()) {
      setError("Title is required.");
      return;
    }
    setError(null);
    setSubmitting(true);
    try {
      const resp = await apiFetch(`/v1/collections/${collectionId}/bookdvd`, {
        method: "POST",
        body: JSON.stringify({
          collectionId,
          title: title.trim(),
          upc: upc.trim(),
          quantity,
          notes,
        }),
      });
      if (!resp.ok) {
        setError("Failed to add item.");
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
        <label htmlFor="title">Title</label>
        <input
          id="title"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Title"
        />
      </div>

      <div className="field">
        <label htmlFor="upc">
          UPC <span style={{ color: "var(--text-muted)", fontWeight: 400 }}>(optional)</span>
        </label>
        <input
          id="upc"
          value={upc}
          onChange={(e) => setUpc(e.target.value)}
          placeholder="Barcode / UPC"
          maxLength={20}
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
