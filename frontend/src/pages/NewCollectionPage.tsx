import { useState } from "react";
import { useNavigate, Link } from "react-router-dom";
import { apiFetch } from "../api/auth";

const COLLECTION_TYPES = [
  { value: "COLLECTION_TYPE_MTG_CARDS", label: "MtG Cards" },
  { value: "COLLECTION_TYPE_MTG_SEALED", label: "MtG Sealed Product" },
  { value: "COLLECTION_TYPE_MTG_DECK", label: "MtG Deck" },
  { value: "COLLECTION_TYPE_BOOKS", label: "Books" },
  { value: "COLLECTION_TYPE_DVDS", label: "DVDs" },
];

export default function NewCollectionPage() {
  const navigate = useNavigate();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [type, setType] = useState(COLLECTION_TYPES[0].value);
  const [isPublic, setIsPublic] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
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
  }

  return (
    <main>
      <Link to="/collections">← Back</Link>
      <h1>New Collection</h1>
      <form onSubmit={handleSubmit}>
        <label>
          Name
          <input value={name} onChange={(e) => setName(e.target.value)} required />
        </label>
        <label>
          Description
          <textarea value={description} onChange={(e) => setDescription(e.target.value)} />
        </label>
        <label>
          Type
          <select value={type} onChange={(e) => setType(e.target.value)}>
            {COLLECTION_TYPES.map((t) => (
              <option key={t.value} value={t.value}>
                {t.label}
              </option>
            ))}
          </select>
        </label>
        <label>
          <input
            type="checkbox"
            checked={isPublic}
            onChange={(e) => setIsPublic(e.target.checked)}
          />
          Make public
        </label>
        {error && <p role="alert">{error}</p>}
        <button type="submit">Create</button>
      </form>
    </main>
  );
}
