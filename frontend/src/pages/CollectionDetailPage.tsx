import { useEffect, useState } from "react";
import { useParams, Link, useNavigate } from "react-router-dom";
import { apiFetch } from "../api/auth";
import { Collection, COLLECTION_TYPE_LABELS, collectionTypeBadgeClass } from "../api/types";
import AppLayout from "../components/AppLayout";
import { useAuth } from "../context/AuthContext";

export default function CollectionDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { token } = useAuth();
  const navigate = useNavigate();
  const [collection, setCollection] = useState<Collection | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    apiFetch(`/v1/collections/${id}`)
      .then(async (r) => {
        if (!r.ok) { setError("Collection not found."); return; }
        const data = await r.json();
        setCollection(data.collection ?? null);
      })
      .finally(() => setLoading(false));
  }, [id]);

  async function handleDelete() {
    if (!confirm("Delete this collection? This cannot be undone.")) return;
    setDeleting(true);
    const resp = await apiFetch(`/v1/collections/${id}`, { method: "DELETE" });
    if (resp.ok) {
      navigate("/collections");
    } else {
      setDeleting(false);
    }
  }

  // Decode user ID from JWT to show owner actions.
  const userId = token ? parseUserId(token) : null;
  const isOwner = collection && userId !== null && String(userId) === collection.ownerId;

  if (loading) return <AppLayout><p className="loading">Loading…</p></AppLayout>;
  if (error || !collection) return <AppLayout><p className="error-page">{error ?? "Collection not found."}</p></AppLayout>;

  return (
    <AppLayout>
      <div className="page">
        <Link to="/collections" className="back-link">← All collections</Link>

        <div className="detail-header">
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", gap: 16 }}>
            <h1>{collection.name}</h1>
            {isOwner && (
              <button className="btn-danger" onClick={handleDelete} disabled={deleting}>
                {deleting ? "Deleting…" : "Delete"}
              </button>
            )}
          </div>
          {collection.description && <p className="desc">{collection.description}</p>}
          <div className="meta">
            <span className={collectionTypeBadgeClass(collection.type)}>
              {COLLECTION_TYPE_LABELS[collection.type]}
            </span>
            {collection.isPublic && <span className="badge badge-public">Public</span>}
          </div>
        </div>

        <div className="page-header" style={{ marginBottom: 16 }}>
          <h2 style={{ margin: 0, fontSize: 16, fontWeight: 600 }}>Items</h2>
          {isOwner && (
            <Link to={`/collections/${id}/add`}>
              <button className="btn-sm">+ Add item</button>
            </Link>
          )}
        </div>

        <div className="empty-state" style={{ padding: "32px 0" }}>
          <p>No items yet.{isOwner ? " Add one to get started." : ""}</p>
        </div>
      </div>
    </AppLayout>
  );
}

function parseUserId(token: string): number | null {
  try {
    const payload = JSON.parse(atob(token.split(".")[1]));
    return payload.uid ?? null;
  } catch {
    return null;
  }
}
