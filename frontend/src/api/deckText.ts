// Client-side previews of deck/card list lines, for bulk import progress.
// Mirrors parseMoxfieldLine, parseDeckLine and parseBoardHeader in
// internal/handler/card_bulk.go and deck_bulk.go; the server is authoritative.

export const MOXFIELD_LINE = /^(\d+)x?\s+(.+?)\s+\(([A-Za-z0-9]+)\)\s+(\S+)(?:\s+\*([FE])\*)?$/;
const NAME_LINE = /^(?:(\d+)x?\s+)?(.+?)(?:\s+\*[FE]\*)?$/;

const BOARD_HEADERS = ["commander", "deck", "main", "mainboard", "sideboard"];

export function isBoardHeader(line: string): boolean {
  return BOARD_HEADERS.includes(line.replace(/:$/, "").trim().toLowerCase());
}

// Describes a deck list line ("4 Lightning Bolt", "Sol Ring", or Moxfield
// format), or null for a section header.
export function describeDeckLine(line: string): { label: string; quantity: number } | null {
  if (isBoardHeader(line)) return null;
  const mox = MOXFIELD_LINE.exec(line);
  if (mox) return { label: `${mox[2]} · ${mox[3].toUpperCase()} #${mox[4]}`, quantity: Number(mox[1]) };
  const m = NAME_LINE.exec(line);
  if (!m) return { label: line, quantity: 1 };
  return { label: m[2], quantity: m[1] ? Number(m[1]) : 1 };
}
