import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { apiFetch } from "../api/auth";
import {
  CardItem,
  Collection,
  COLLECTION_TYPE_LABELS,
  collectionTypeBadgeClass,
  FOIL_TREATMENT_LABELS,
} from "../api/types";
import AppLayout from "../components/AppLayout";

export default function CollectionsPage() {
  const [collections, setCollections] = useState<Collection[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    apiFetch("/v1/collections")
      .then((r) => r.json())
      .then((data) => {
        // The default collection first, then newest first as the API returns them.
        const all: Collection[] = data.collections ?? [];
        setCollections([...all.filter((c) => c.isDefault), ...all.filter((c) => !c.isDefault)]);
      })
      .finally(() => setLoading(false));
  }, []);

  const [query, setQuery] = useState("");
  // Matching cards keyed by collection id; null when no search is active.
  const [matches, setMatches] = useState<Map<string, CardItem[]> | null>(null);
  const [searching, setSearching] = useState(false);

  useEffect(() => {
    const q = query.trim();
    if (q === "") {
      setMatches(null);
      setSearching(false);
      return;
    }
    setSearching(true);
    const controller = new AbortController();
    const timer = setTimeout(() => {
      apiFetch(`/v1/search/cards?query=${encodeURIComponent(q)}`, { signal: controller.signal })
        .then((r) => r.json())
        .then((data: { matches?: { collectionId: string; items?: CardItem[] }[] }) => {
          const byCollection = new Map<string, CardItem[]>();
          for (const m of data.matches ?? []) byCollection.set(m.collectionId, m.items ?? []);
          setMatches(byCollection);
          setSearching(false);
        })
        .catch((err) => {
          if (err.name !== "AbortError") setSearching(false);
        });
    }, 250);
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [query]);

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
            <p>Create your first collection to start tracking your cards, sealed product, books, or other items.</p>
            <Link to="/collections/new">
              <button className="btn-sm">Create a collection</button>
            </Link>
          </div>
        ) : (
          <>
            <div className="collection-search">
              <input
                type="search"
                placeholder="Search cards across collections…"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                aria-label="Search cards by name"
              />
              {matches && !searching && (
                <span className="collection-search-status">
                  {matches.size === 0
                    ? "No matching cards"
                    : `Found in ${matches.size} collection${matches.size === 1 ? "" : "s"}`}
                </span>
              )}
            </div>
            <div className="collection-grid">
              {collections.map((c) => {
                const cardMatches = matches?.get(c.id);
                const dimmed = matches !== null && !cardMatches;
                return (
                  <Link
                    key={c.id}
                    to={`/collections/${c.id}`}
                    className={`collection-card${dimmed ? " collection-card-dimmed" : ""}`}
                  >
                    <h2>{c.name}</h2>
                    {cardMatches && <SearchMatches items={cardMatches} />}
                    {c.description && <p className="card-desc">{c.description}</p>}
                    <div className="card-meta">
                      <span className={collectionTypeBadgeClass(c.type)}>
                        {COLLECTION_TYPE_LABELS[c.type]}
                      </span>
                      {c.isDefault && <span className="badge">Default</span>}
                      {c.isPublic && <span className="badge badge-public">Public</span>}
                    </div>
                  </Link>
                );
              })}
            </div>
          </>
        )}
      </div>
    </AppLayout>
  );
}

const MAX_MATCHES_SHOWN = 8;

function SearchMatches({ items }: { items: CardItem[] }) {
  const shown = items.slice(0, MAX_MATCHES_SHOWN);
  return (
    <ul className="search-matches">
      {shown.map((item) => (
        <li key={item.id}>
          <span className="search-match-qty">{item.quantity}×</span>
          <span className="search-match-name">{item.cardName}</span>
          <span className="search-match-meta">
            {item.setCode.toUpperCase()}
            {item.foilTreatment !== "FOIL_TREATMENT_NONFOIL" &&
              ` · ${FOIL_TREATMENT_LABELS[item.foilTreatment]}`}
            {item.language && item.language !== "en" && ` · ${item.language.toUpperCase()}`}
          </span>
          {item.checkouts?.map((c) => (
            <span key={c.deckId} className="search-match-meta card-checkouts">{c.quantity} in {c.deckName}</span>
          ))}
        </li>
      ))}
      {items.length > shown.length && (
        <li className="search-match-more">+{items.length - shown.length} more</li>
      )}
    </ul>
  );
}
