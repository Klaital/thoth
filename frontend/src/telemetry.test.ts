import { describe, expect, it } from "vitest";
import { routeName, userIdFromToken } from "./telemetry";

describe("routeName", () => {
  it("replaces numeric segments", () => {
    expect(routeName("/collections/12/decks/345")).toBe("/collections/:id/decks/:id");
    expect(routeName("/decks/7")).toBe("/decks/:id");
  });
  it("leaves other paths alone", () => {
    expect(routeName("/collections")).toBe("/collections");
    expect(routeName("/")).toBe("/");
  });
});

describe("userIdFromToken", () => {
  const b64url = (o: object) =>
    btoa(JSON.stringify(o)).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");

  it("reads the uid claim", () => {
    expect(userIdFromToken(`x.${b64url({ uid: 42, email: "a@b.c" })}.sig`)).toBe("42");
  });
  it("returns null for missing or malformed tokens", () => {
    expect(userIdFromToken(null)).toBeNull();
    expect(userIdFromToken("garbage")).toBeNull();
  });
});
