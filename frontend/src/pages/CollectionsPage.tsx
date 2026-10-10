import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { apiFetch } from "../api/auth";

interface Collection {
  id: string;
  name: string;
  description: string;
  type: string;
  isPublic: boolean;
}

export default function CollectionsPage() {
  const [collections, setCollections] = useState<Collection[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    apiFetch("/v1/collections")
      .then((r) => r.json())
      .then((data) => setCollections(data.collections ?? []))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <p>Loading…</p>;

  return (
    <main>
      <h1>My Collections</h1>
      <Link to="/collections/new">New collection</Link>
      <ul>
        {collections.map((c) => (
          <li key={c.id}>
            <Link to={`/collections/${c.id}`}>{c.name}</Link>
            {" — "}
            {c.type}
          </li>
        ))}
      </ul>
      {collections.length === 0 && <p>No collections yet. Create one!</p>}
    </main>
  );
}
