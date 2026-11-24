import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { setTokens } from "../api/auth";
import { DeckOwnership } from "../api/types";
import { AuthProvider } from "../context/AuthContext";
import DeckDetailPage from "./DeckDetailPage";

// A token for user 1; only the payload is read client-side.
const token = `x.${btoa(JSON.stringify({ uid: 1 }))}.x`;
let ownership: DeckOwnership;

beforeEach(() => {
  localStorage.clear();
  setTokens(token, "r");
  vi.stubGlobal("fetch", vi.fn(async (url: string) => {
    if (url === "/v1/decks/7") return Response.json({ deck: { id: "7", ownerId: "1", name: "Atraxa", notes: "", isPublic: false } });
    if (url === "/v1/decks/7/cards") {
      return Response.json({ cards: [{ id: "1", deckId: "7", cardName: "City of Brass", board: "DECK_BOARD_MAIN", quantity: 1 }] });
    }
    if (url === "/v1/decks/7/ownership") return Response.json(ownership);
    return new Response("not found", { status: 404 });
  }));
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

function renderDeck() {
  render(
    <MemoryRouter initialEntries={["/decks/7"]}>
      <AuthProvider>
        <Routes><Route path="/decks/:id" element={<DeckDetailPage />} /></Routes>
      </AuthProvider>
    </MemoryRouter>,
  );
}

test("the badge follows a single copy through check-out and return", async () => {
  // Owned, still in its box (also the state after returning it).
  ownership = { cards: [{ cardName: "City of Brass", needed: 1, free: 1, checkedOutHere: 0, missing: 0 }], totalMissing: 0, totalCheckedOut: 0 };
  renderDeck();
  // A shortcut into the check-out wizard for just this card.
  expect((await screen.findByText("0/1 checked out")).closest("a")?.getAttribute("href"))
    .toBe("/decks/7/checkout?card=City%20of%20Brass");
  cleanup();

  // Checked out to the deck.
  ownership = { cards: [{ cardName: "City of Brass", needed: 1, free: 0, checkedOutHere: 1, missing: 0 }], totalMissing: 0, totalCheckedOut: 1 };
  renderDeck();
  // Fully checked out: the shortcut starts a return instead.
  expect((await screen.findByText("1/1 checked out")).closest("a")?.getAttribute("href"))
    .toBe("/decks/7/return?card=City%20of%20Brass");
});

test("a card you don't own says how many are missing", async () => {
  ownership = { cards: [{ cardName: "City of Brass", needed: 1, missing: 1 }], totalMissing: 1 };
  renderDeck();
  const badge = await screen.findByTitle("0 checked out, 1 missing");
  expect(badge.textContent).toBe("0/1 checked out · 1 missing");
  // Nothing to check out or return, so it isn't a link.
  expect(badge.tagName).toBe("SPAN");
});
