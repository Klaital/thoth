import { ReactNode, useState } from "react";
import { apiFetch } from "../api/auth";
import { BulkImportEntry, BulkImportLineResult } from "../api/types";

interface Props {
  collectionId: number;
  // Path under the collection, e.g. "sealed/bulk".
  endpoint: string;
  placeholder: string;
  help: ReactNode;
  // Extra fields rendered above the actions, and the values they add to the request.
  extraFields?: ReactNode;
  extraBody?: Record<string, unknown>;
  // Describes a non-blank line for the progress list, or returns null if it
  // isn't an item (e.g. a section header). Defaults to the line itself.
  describeLine?: (line: string) => { label: string; quantity: number } | null;
  // Called with every line's status when an import starts and as each line finishes.
  onProgress?: (entries: BulkImportEntry[]) => void;
  onSuccess: () => void;
  onItemAdded?: () => void;
}

// One JSON object from the response. grpc-gateway sends a server-streaming RPC
// as one {"result"} or {"error"} object per line, and a unary RPC as a single
// {"results"} object, or {"message"} on failure.
interface ResponseChunk {
  result?: BulkImportLineResult<unknown>;
  results?: BulkImportLineResult<unknown>[];
  error?: { message?: string };
  message?: string;
}

async function* readJsonLines(resp: Response): AsyncGenerator<ResponseChunk> {
  if (!resp.body) return;
  const reader = resp.body.getReader();
  const decoder = new TextDecoder();
  let buf = "";
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    buf += decoder.decode(value, { stream: true });
    let nl: number;
    while ((nl = buf.indexOf("\n")) >= 0) {
      const line = buf.slice(0, nl).trim();
      buf = buf.slice(nl + 1);
      if (line) yield JSON.parse(line);
    }
  }
  buf += decoder.decode();
  if (buf.trim()) yield JSON.parse(buf);
}

export default function BulkImportForm({
  collectionId,
  endpoint,
  placeholder,
  help,
  extraFields,
  extraBody,
  describeLine = (line) => ({ label: line, quantity: 1 }),
  onProgress,
  onSuccess,
  onItemAdded,
}: Props) {
  const [text, setText] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [entries, setEntries] = useState<BulkImportEntry[] | null>(null);

  async function handleFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setText(await file.text());
    setEntries(null);
    e.target.value = "";
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);

    const lines = text.split("\n").map((l) => l.trim());
    const byLine = new Map<number, BulkImportEntry>();
    lines.forEach((line, i) => {
      if (!line) return;
      const desc = describeLine(line);
      if (desc) byLine.set(i + 1, { lineNumber: i + 1, line, ...desc, status: "pending" });
    });
    const publish = () => {
      const snapshot = [...byLine.values()];
      setEntries(snapshot);
      onProgress?.(snapshot);
    };
    publish();

    let fatal: string | null = null;
    try {
      const resp = await apiFetch(`/v1/collections/${collectionId}/${endpoint}`, {
        method: "POST",
        body: JSON.stringify({ ...extraBody, collectionId, text }),
      });
      if (!resp.ok) fatal = "Bulk import failed.";
      for await (const chunk of readJsonLines(resp)) {
        if (chunk.error) fatal = chunk.error.message ?? "Bulk import failed.";
        else if (!resp.ok && chunk.message) fatal = chunk.message;
        const results = chunk.results ?? (chunk.result ? [chunk.result] : []);
        for (const r of results) {
          const entry = byLine.get(r.lineNumber);
          if (!entry) continue;
          byLine.set(r.lineNumber, {
            ...entry,
            status: r.error ? "failed" : "added",
            item: r.item,
            warning: r.warning,
            error: r.error,
          });
        }
        if (results.length > 0) publish();
      }
    } catch {
      fatal = "Bulk import failed.";
    }

    // Anything without a result wasn't imported (e.g. the request failed).
    for (const [n, entry] of byLine) {
      if (entry.status === "pending") byLine.set(n, { ...entry, status: "failed", error: fatal ?? "Not imported." });
    }
    publish();
    setError(fatal);

    // Keep only the lines that weren't added so they can be fixed and
    // resubmitted, along with the section headers they sit under.
    const kept: string[] = [];
    let header: string | null = null;
    lines.forEach((line, i) => {
      if (!line) return;
      const entry = byLine.get(i + 1);
      if (!entry) {
        header = line;
        return;
      }
      if (entry.status === "added") return;
      if (header) {
        kept.push(header);
        header = null;
      }
      kept.push(line);
    });
    setText(kept.join("\n"));
    setSubmitting(false);
    onItemAdded?.();
  }

  const doneCount = entries?.filter((e) => e.status !== "pending").length ?? 0;
  const addedCount = entries?.filter((e) => e.status === "added").length ?? 0;

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
          disabled={submitting}
          style={{ resize: "vertical", fontFamily: "monospace", fontSize: "0.85rem" }}
        />
        <p style={{ margin: "0.25rem 0 0", fontSize: "0.82rem", color: "var(--text-muted)" }}>{help}</p>
      </div>

      <div className="field">
        <label htmlFor="bulk-file">
          Or load a file <span style={{ color: "var(--text-muted)", fontWeight: 400 }}>(.txt)</span>
        </label>
        <input id="bulk-file" type="file" accept=".txt,text/plain" onChange={handleFile} disabled={submitting} />
      </div>

      {extraFields}

      {error && <p className="error-msg" role="alert">{error}</p>}

      {entries && !submitting && (
        <p className="field" role="status" style={{ margin: "0 0 1rem" }}>
          Added {addedCount} of {entries.length} line{entries.length === 1 ? "" : "s"}.
          {addedCount < entries.length && " Lines that weren't added are left above to fix and retry."}
        </p>
      )}

      <div className="form-actions">
        <button type="submit" className="btn-primary" disabled={submitting || !text.trim()}>
          {submitting ? `Importing… ${doneCount} of ${entries?.length ?? 0}` : "Import"}
        </button>
        <button type="button" className="btn-secondary" onClick={onSuccess}>
          Done
        </button>
      </div>
    </form>
  );
}
