import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { DeckCheckout } from "../api/types";
import { AuthProvider } from "../context/AuthContext";
import DeckReturnPage from "./DeckReturnPage";

const co = (id: string, cardName: string, collectionName: string): DeckCheckout => ({
  collectionCardId: id, collectionId: id, collectionName, cardName, scryfallCardId: `p${id}`,
  setCode: "mh2", collectorNumber: "1", foilTreatment: "FOIL_TREATMENT_NONFOIL", language: "en", quantity: 1,
});

let posted: unknown[] = [];

beforeEach(() => {
  posted = [];
  localStorage.clear();
  vi.stubGlobal("fetch", vi.fn(async (url: string, init?: RequestInit) => {
    if (url === "/v1/decks/7") return Response.json({ deck: { id: "7", ownerId: "1", name: "Atraxa", notes: "", isPublic: false } });
    if (url === "/v1/decks/7/checkouts") {
      return Response.json({ checkouts: [co("11", "City of Brass", "Box A"), co("22", "Sol Ring", "Box B")] });
    }
    if (url === "/v1/decks/7/return" && init?.method === "POST") {
      posted.push(JSON.parse(String(init.body)));
      return Response.json({});
    }
    return new Response("not found", { status: 404 });
  }));
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

test("?card= returns just that card", async () => {
  render(
    <MemoryRouter initialEntries={["/decks/7/return?card=Sol%20Ring"]}>
      <AuthProvider>
        <Routes>
          <Route path="/decks/:id/return" element={<DeckReturnPage />} />
          <Route path="/decks/:id" element={<p>deck page</p>} />
        </Routes>
      </AuthProvider>
    </MemoryRouter>,
  );

  expect(await screen.findByRole("heading", { name: "Return Sol Ring" })).toBeTruthy();
  expect(screen.queryByText(/City of Brass/)).toBeNull();
  expect(screen.getByRole("link", { name: /whole deck/ }).getAttribute("href")).toBe("/decks/7/return");

  const boxes = screen.getAllByRole("checkbox");
  expect(boxes).toHaveLength(1);
  fireEvent.click(boxes[0]);
  fireEvent.click(screen.getByRole("button", { name: "Return 1 card" }));
  expect(await screen.findByText("deck page")).toBeTruthy();
  expect(posted).toEqual([{ picks: [{ collectionCardId: "22", quantity: 1 }] }]);
});
