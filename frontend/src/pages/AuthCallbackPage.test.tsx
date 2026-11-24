import { StrictMode } from "react";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { BrowserRouter } from "react-router-dom";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import App from "../App";
import { getAccessToken, getRefreshToken } from "../api/auth";
import { AuthProvider } from "../context/AuthContext";

// Renders the whole app as main.tsx does (StrictMode + BrowserRouter), so the
// test follows the real post-login redirects, starting from the URL the
// backend's /auth/google/callback redirects the browser to.
function renderAppAt(url: string) {
  window.history.replaceState(null, "", url);
  return render(
    <StrictMode>
      <BrowserRouter>
        <AuthProvider>
          <App />
        </AuthProvider>
      </BrowserRouter>
    </StrictMode>,
  );
}

beforeEach(() => {
  localStorage.clear();
  // The collections page loads collections once signed in.
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => new Response(JSON.stringify({ collections: [] }), { status: 200 })),
  );
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

test("Google sign-in callback stores the tokens and lands on the collections page", async () => {
  renderAppAt("/auth/callback?access_token=access-123&refresh_token=refresh-456");

  expect(await screen.findByRole("heading", { name: "My Collections" })).toBeTruthy();
  // Wait out any further navigation (StrictMode re-runs effects) before
  // checking where we ended up.
  await waitFor(() => expect(window.location.pathname).toBe("/collections"));
  expect(screen.queryByRole("button", { name: /continue with google/i })).toBeNull();
  // Tokens are stored, and no longer sitting in the address bar.
  expect(getAccessToken()).toBe("access-123");
  expect(getRefreshToken()).toBe("refresh-456");
  expect(window.location.search).toBe("");
  // The signed-in request used the new token.
  expect(fetch).toHaveBeenCalledWith(
    "/v1/collections",
    expect.objectContaining({ headers: expect.objectContaining({ Authorization: "Bearer access-123" }) }),
  );
});

test("callback without tokens goes back to the login page", async () => {
  renderAppAt("/auth/callback?error=access_denied");

  expect(await screen.findByRole("button", { name: /continue with google/i })).toBeTruthy();
  expect(window.location.pathname).toBe("/login");
  expect(getAccessToken()).toBeNull();
});
