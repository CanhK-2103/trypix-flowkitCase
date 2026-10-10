export type Finding = {
  code: string;
  message: string;
  path: string;
  node_id?: string;
};
export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
    public findings: Finding[] = [],
  ) {
    super(message);
  }
}
export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response;
  try {
    response = await fetch(path, {
      ...init,
      headers: { "Content-Type": "application/json", ...init?.headers },
    });
  } catch {
    throw new ApiError(
      0,
      "NETWORK_ERROR",
      "API bağlantısı kurulamadı. Aynı isteği tekrar deneyebilirsiniz.",
    );
  }
  if (response.status === 204) return undefined as T;
  const body = await response.json();
  if (!response.ok)
    throw new ApiError(
      response.status,
      body.error?.code ?? "API_ERROR",
      body.error?.message ?? "İstek başarısız",
      body.error?.findings ?? [],
    );
  return body;
}
export function retryGet(count: number, error: Error) {
  return (
    count < 1 &&
    (!(error instanceof ApiError) || error.status === 0 || error.status >= 500)
  );
}
export function active(status?: string) {
  return status === "queued" || status === "running";
}
export function pollInterval(status?: string) {
  return active(status) ? 2000 : false;
}
export type Page<T> = { items: T[]; next_cursor: string };
export async function allPages<T>(path: string): Promise<T[]> {
  const items: T[] = [];
  let cursor = "";
  do {
    const page = await api<Page<T>>(
      `${path}${path.includes("?") ? "&" : "?"}limit=100${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`,
    );
    items.push(...page.items);
    cursor = page.next_cursor;
  } while (cursor);
  return items;
}
