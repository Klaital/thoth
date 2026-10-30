import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { apiFetch } from "../api/auth";
import { Collection, COLLECTION_TYPE_LABELS, collectionTypeBadgeClass } from "../api/types";
import AppLayout from "../components/AppLayout";
import CommanderSummary from "../components/CommanderSummary";

export default function CollectionsPage() {
  const [collections, setCollections] = useState<Collection[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    apiFetch("/v1/collections")
      .then((r) => r.json())
      .then((data) => setCollections(data.collections ?? []))
      .finally(() => setLoading(false));
  }, []);

  return (
    <AppLayout>
      <div className="page">
        <div className="page-header">
          <h1>My Collections</h1>
          <Link to="/collections/new">
            <button className="btn-sm">+ New Collection</button>
          </Link>
        </div>

        {loading ? (
          <p className="loading">Loading…</p>
        ) : collections.length === 0 ? (
          <div className="empty-state">
            <h2>No collections yet</h2>
            <p>Create your first collection to start tracking your cards, books, or other items.</p>
            <Link to="/collections/new">
              <button className="btn-sm">Create a collection</button>
            </Link>
          </div>
        ) : (
          <div className="collection-grid">
            {collections.map((c) => (
              <Link key={c.id} to={`/collections/${c.id}`} className="collection-card">
                <h2>{c.name}</h2>
                {c.commanders && c.commanders.length > 0 && <CommanderSummary commanders={c.commanders} />}
                {c.description && <p className="card-desc">{c.description}</p>}
                <div className="card-meta">
                  <span className={collectionTypeBadgeClass(c.type)}>
                    {COLLECTION_TYPE_LABELS[c.type]}
                  </span>
                  {c.isPublic && <span className="badge badge-public">Public</span>}
                </div>
              </Link>
            ))}
          </div>
        )}
      </div>
    </AppLayout>
  );
}
