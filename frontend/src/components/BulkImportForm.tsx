import { ReactNode, useState } from "react";
import { apiFetch } from "../api/auth";
import { BulkImportLineResult } from "../api/types";

interface Props {
  collectionId: number;
  // Path under the collection, e.g. "sealed/bulk".
  endpoint: string;
  placeholder: string;
  help: ReactNode;
  // Extra fields rendered above the actions, and the values they add to the request.
  extraFields?: ReactNode;
  extraBody?: Record<string, unknown>;
  onSuccess: () => void;
  onItemAdded?: () => void;
}

export default function BulkImportForm({
  collectionId,
  endpoint,
  placeholder,
  help,
  extraFields,
  extraBody,
  onSuccess,
  onItemAdded,
}: Props) {
  const [text, setText] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [results, setResults] = useState<BulkImportLineResult<unknown>[] | null>(null);

  async function handleFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setText(await file.text());
    setResults(null);
    e.target.value = "";
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      const resp = await apiFetch(`/v1/collections/${collectionId}/${endpoint}`, {
        method: "POST",
        body: JSON.stringify({ ...extraBody, collectionId, text }),
      });
      if (!resp.ok) {
        const data = await resp.json().catch(() => null);
        setError(data?.message ?? "Bulk import failed.");
        return;
      }
      const data = await resp.json();
      const lineResults: BulkImportLineResult<unknown>[] = data.results ?? [];
      setResults(lineResults);
      // Keep only the lines that failed so they can be fixed and resubmitted.
      setText(lineResults.filter((r) => r.error).map((r) => r.line).join("\n"));
      onItemAdded?.();
    } finally {
      setSubmitting(false);
    }
  }

  const addedCount = results?.filter((r) => r.item).length ?? 0;
  const issues = results?.filter((r) => r.warning || r.error) ?? [];

  return (
    <form onSubmit={handleSubmit}>
      <div className="field">
        <label htmlFor="bulk-text">One item per line</label>
        <textarea
          id="bulk-text"
          rows={12}
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder={placeholder}
          style={{ resize: "vertical", fontFamily: "monospace", fontSize: "0.85rem" }}
        />
        <p style={{ margin: "0.25rem 0 0", fontSize: "0.82rem", color: "var(--text-muted)" }}>{help}</p>
      </div>

      <div className="field">
        <label htmlFor="bulk-file">
          Or load a file <span style={{ color: "var(--text-muted)", fontWeight: 400 }}>(.txt)</span>
        </label>
        <input id="bulk-file" type="file" accept=".txt,text/plain" onChange={handleFile} />
      </div>

      {extraFields}

      {error && <p className="error-msg" role="alert">{error}</p>}

      {results && (
        <div className="field" role="status">
          <p style={{ margin: "0 0 0.25rem" }}>
            Added {addedCount} of {results.length} line{results.length === 1 ? "" : "s"}.
          </p>
          {issues.length > 0 && (
            <ul style={{ margin: 0, paddingLeft: "1.1rem", fontSize: "0.82rem" }}>
              {issues.map((r) => (
                <li
                  key={r.lineNumber}
                  style={{ color: r.error ? "var(--danger)" : "var(--text-muted)" }}
                >
                  Line {r.lineNumber} · {r.name || r.line}: {r.error || r.warning}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}

      <div className="form-actions">
        <button type="submit" className="btn-primary" disabled={submitting || !text.trim()}>
          {submitting ? "Importing…" : "Import"}
        </button>
        <button type="button" className="btn-secondary" onClick={onSuccess}>
          Done
        </button>
      </div>
    </form>
  );
}
