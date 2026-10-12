import { useEffect, useState } from "react";
import { useParams, Link, useNavigate } from "react-router-dom";
import { apiFetch } from "../api/auth";
import { Collection } from "../api/types";
import AppLayout from "../components/AppLayout";
import AddCardItemForm from "../components/AddCardItemForm";
import AddSealedItemForm from "../components/AddSealedItemForm";
import AddBookDvdItemForm from "../components/AddBookDvdItemForm";

export default function AddItemPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [collection, setCollection] = useState<Collection | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    apiFetch(`/v1/collections/${id}`)
      .then((r) => r.json())
      .then((data) => setCollection(data.collection ?? null))
      .finally(() => setLoading(false));
  }, [id]);

  function onSuccess() {
    navigate(`/collections/${id}`);
  }

  if (loading) return <AppLayout><p className="loading">Loading…</p></AppLayout>;
  if (!collection) return <AppLayout><p className="error-page">Collection not found.</p></AppLayout>;

  const collectionId = Number(id);

  return (
    <AppLayout>
      <div className="page">
        <Link to={`/collections/${id}`} className="back-link">← Back to {collection.name}</Link>
        <div className="form-card">
          <h1>Add Item</h1>
          {(collection.type === "COLLECTION_TYPE_MTG_CARDS" || collection.type === "COLLECTION_TYPE_MTG_DECK") && (
            <AddCardItemForm collectionId={collectionId} onSuccess={onSuccess} />
          )}
          {collection.type === "COLLECTION_TYPE_MTG_SEALED" && (
            <AddSealedItemForm collectionId={collectionId} onSuccess={onSuccess} />
          )}
          {(collection.type === "COLLECTION_TYPE_BOOKS" || collection.type === "COLLECTION_TYPE_DVDS") && (
            <AddBookDvdItemForm collectionId={collectionId} onSuccess={onSuccess} />
          )}
        </div>
      </div>
    </AppLayout>
  );
}
