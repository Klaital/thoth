// Error message from a gRPC-gateway error response, if any.
export async function errorMessage(resp: Response, fallback: string): Promise<string> {
  try {
    const data = await resp.json();
    return data.message || fallback;
  } catch {
    return fallback;
  }
}
