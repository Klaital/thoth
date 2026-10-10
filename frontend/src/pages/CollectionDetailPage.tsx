import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { apiFetch } from "../api/auth";

interface Collection {
  id: string;
  name: string;
  description: string;
  type: string;
}

export default function CollectionDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [collection, setCollection] = useState<Collection | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    apiFetch(`/v1/collections/${id}`)
      .then((r) => r.json())
      .then((data) => setCollection(data.collection ?? null))
      .finally(() => setLoading(false));
  }, [id]);

  if (loading) return <p>Loading…</p>;
  if (!collection) return <p>Collection not found.</p>;

  return (
    <main>
      <Link to="/collections">← Back</Link>
      <h1>{collection.name}</h1>
      <p>{collection.description}</p>
      <p>Type: {collection.type}</p>
      <Link to={`/collections/${id}/add`}>Add item</Link>
      {/* TODO: render items list based on collection.type */}
    </main>
  );
}
