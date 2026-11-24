import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { DeckOwnership } from "../api/types";
import { AuthProvider } from "../context/AuthContext";
import DeckCheckoutPage from "./DeckCheckoutPage";

const ownership: DeckOwnership = {
  cards: [
    {
      cardName: "City of Brass",
      needed: 2,
      free: 3,
      copies: [
        {
          collectionCardId: "11", collectionId: "1", collectionName: "Box A", scryfallCardId: "p1",
          setCode: "mh2", collectorNumber: "259", foilTreatment: "FOIL_TREATMENT_NONFOIL", language: "en",
          quantity: 1, free: 1, suggestedQuantity: 1,
        },
        {
          collectionCardId: "22", collectionId: "2", collectionName: "Box B", scryfallCardId: "p1",
          setCode: "mh2", collectorNumber: "259", foilTreatment: "FOIL_TREATMENT_NONFOIL", language: "ja",
          quantity: 2, free: 2, suggestedQuantity: 1,
        },
      ],
    },
    { cardName: "Swords to Plowshares", needed: 1, missing: 1 },
  ],
  totalMissing: 1,
};

let posted: unknown[] = [];

beforeEach(() => {
  posted = [];
  localStorage.clear();
  vi.stubGlobal("fetch", vi.fn(async (url: string, init?: RequestInit) => {
    if (url === "/v1/decks/7") return Response.json({ deck: { id: "7", ownerId: "1", name: "Atraxa", notes: "", isPublic: false } });
    if (url === "/v1/decks/7/ownership") return Response.json(ownership);
    if (url === "/v1/decks/7/checkout" && init?.method === "POST") {
      posted.push(JSON.parse(String(init.body)));
      return Response.json({});
    }
    if (url === "/v1/decks/7/cards") return Response.json({ cards: [] });
    return new Response("not found", { status: 404 });
  }));
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

test("check-out records only the cards ticked off in the pull list", async () => {
  render(
    <MemoryRouter initialEntries={["/decks/7/checkout"]}>
      <AuthProvider>
        <Routes>
          <Route path="/decks/:id/checkout" element={<DeckCheckoutPage />} />
          <Route path="/decks/:id" element={<p>deck page</p>} />
        </Routes>
      </AuthProvider>
    </MemoryRouter>,
  );

  // Step 1: suggestions are pre-filled; missing cards are listed as skipped.
  expect(await screen.findByRole("heading", { name: "Check out Atraxa" })).toBeTruthy();
  expect(screen.getByText(/Missing, skipped/)).toBeTruthy();
  const boxB = screen.getByLabelText(/from Box B/) as HTMLInputElement;
  expect(boxB.value).toBe("1");

  // Taking a third copy is more than the deck needs.
  fireEvent.change(boxB, { target: { value: "2" } });
  expect(screen.getByRole("alert").textContent).toMatch(/only needs 2 more/);
  expect((screen.getByRole("button", { name: /Next/ }) as HTMLButtonElement).disabled).toBe(true);
  fireEvent.change(boxB, { target: { value: "1" } });
  fireEvent.click(screen.getByRole("button", { name: "Next: pull 2 cards" }));

  // Step 2: grouped box by box; nothing is ticked yet.
  expect(screen.getByRole("heading", { name: "Box A" })).toBeTruthy();
  expect(screen.getByRole("heading", { name: "Box B" })).toBeTruthy();
  const confirm = screen.getByRole("button", { name: /Check out 0 cards/ }) as HTMLButtonElement;
  expect(confirm.disabled).toBe(true);

  // Pull only the Box B copy: a partial check-out.
  const lines = screen.getAllByRole("checkbox");
  fireEvent.click(lines[1]);
  expect(screen.getByRole("status").textContent).toMatch(/1 of 2 pulled/);
  fireEvent.click(screen.getByRole("button", { name: "Check out 1 card" }));

  expect(await screen.findByText("deck page")).toBeTruthy();
  expect(posted).toEqual([{ picks: [{ collectionCardId: "22", quantity: 1 }] }]);
});

test("?card= limits the wizard to one card", async () => {
  render(
    <MemoryRouter initialEntries={["/decks/7/checkout?card=City%20of%20Brass"]}>
      <AuthProvider>
        <Routes>
          <Route path="/decks/:id/checkout" element={<DeckCheckoutPage />} />
          <Route path="/decks/:id" element={<p>deck page</p>} />
        </Routes>
      </AuthProvider>
    </MemoryRouter>,
  );

  expect(await screen.findByRole("heading", { name: "Check out City of Brass" })).toBeTruthy();
  // The other deck card is neither listed nor reported missing.
  expect(screen.queryByText(/Missing, skipped/)).toBeNull();
  expect(screen.queryByText(/Swords to Plowshares/)).toBeNull();
  expect(screen.getByRole("link", { name: /whole deck/ }).getAttribute("href")).toBe("/decks/7/checkout");

  fireEvent.click(screen.getByRole("button", { name: "Next: pull 2 cards" }));
  for (const box of screen.getAllByRole("checkbox")) fireEvent.click(box);
  fireEvent.click(screen.getByRole("button", { name: "Check out 2 cards" }));
  expect(await screen.findByText("deck page")).toBeTruthy();
  expect(posted).toEqual([{ picks: [{ collectionCardId: "11", quantity: 1 }, { collectionCardId: "22", quantity: 1 }] }]);
});
