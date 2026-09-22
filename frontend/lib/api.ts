export const SESSION_KEY = "pvmoney.session";

async function parse<T>(res: Response): Promise<T> {
  if (res.status === 204) return undefined as T;
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error((data as { error?: string }).error || "server");
  }
  return data as T;
}

function headers(extra?: HeadersInit): Headers {
  const h = new Headers(extra);
  if (typeof window !== "undefined") {
    const token = localStorage.getItem(SESSION_KEY);
    if (token && !h.has("Authorization")) h.set("Authorization", `Bearer ${token}`);
  }
  return h;
}

async function request<T>(url: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(url, {
    credentials: "include",
    ...init,
    headers: headers(init.headers),
  });
  if (res.status === 401 && typeof window !== "undefined" && !url.startsWith("/api/auth/")) {
    window.dispatchEvent(new Event("pvmoney:unauthorized"));
  }
  return parse<T>(res);
}

const json = { "Content-Type": "application/json" };

export const api = {
  authStatus: () => request<import("./types").AuthStatus>("/api/auth/status"),
  registerStart: (body: unknown) =>
    request<import("./types").AuthChallenge>("/api/auth/register/start", {
      method: "POST",
      headers: json,
      body: JSON.stringify(body),
    }),
  loginStart: (body: unknown) =>
    request<import("./types").AuthChallenge>("/api/auth/login/start", {
      method: "POST",
      headers: json,
      body: JSON.stringify(body),
    }),
  challenge: (id: string) => request<import("./types").AuthChallenge>(`/api/auth/challenge/${id}`),
  verifyAuth: (body: unknown) =>
    request<{ token: string; user: import("./types").AuthUser }>("/api/auth/verify", {
      method: "POST",
      headers: json,
      body: JSON.stringify(body),
    }),
  resendAuth: (body: unknown) =>
    request<import("./types").AuthChallenge>("/api/auth/resend", {
      method: "POST",
      headers: json,
      body: JSON.stringify(body),
    }),
  logout: () => request("/api/auth/logout", { method: "POST" }),
  dashboard: () => request<import("./types").Dashboard>("/api/dashboard"),
  accounts: () => request<import("./types").Account[]>("/api/accounts"),
  projects: () => request<import("./types").Project[]>("/api/projects"),
  project: (id: string) => request<import("./types").Project>(`/api/projects/${id}`),
  transactions: (q = "") => request<import("./types").Tx[]>(`/api/transactions${q}`),
  createAccount: (body: unknown) => request("/api/accounts", { method: "POST", headers: json, body: JSON.stringify(body) }),
  updateAccount: (id: string, body: unknown) => request(`/api/accounts/${id}`, { method: "PUT", headers: json, body: JSON.stringify(body) }),
  deleteAccount: (id: string) => request(`/api/accounts/${id}`, { method: "DELETE" }),
  adjustAccount: (id: string, body: unknown) =>
    request(`/api/accounts/${id}/adjust`, { method: "POST", headers: json, body: JSON.stringify(body) }),
  createProject: (body: unknown) => request("/api/projects", { method: "POST", headers: json, body: JSON.stringify(body) }),
  updateProject: (id: string, body: unknown) => request(`/api/projects/${id}`, { method: "PUT", headers: json, body: JSON.stringify(body) }),
  deleteProject: (id: string) => request(`/api/projects/${id}`, { method: "DELETE" }),
  contribute: (id: string, body: unknown) =>
    request(`/api/projects/${id}/contribute`, { method: "POST", headers: json, body: JSON.stringify(body) }),
  withdraw: (id: string, body: unknown) =>
    request(`/api/projects/${id}/withdraw`, { method: "POST", headers: json, body: JSON.stringify(body) }),
  createItem: (projectId: string, body: unknown) =>
    request(`/api/projects/${projectId}/items`, { method: "POST", headers: json, body: JSON.stringify(body) }),
  updateItem: (projectId: string, itemId: string, body: unknown) =>
    request(`/api/projects/${projectId}/items/${itemId}`, { method: "PUT", headers: json, body: JSON.stringify(body) }),
  deleteItem: (projectId: string, itemId: string) => request(`/api/projects/${projectId}/items/${itemId}`, { method: "DELETE" }),
  payItem: (projectId: string, itemId: string, body: unknown) =>
    request(`/api/projects/${projectId}/items/${itemId}/pay`, { method: "POST", headers: json, body: JSON.stringify(body) }),
  convertAsset: (projectId: string, body: unknown) =>
    request(`/api/projects/${projectId}/convert-asset`, { method: "POST", headers: json, body: JSON.stringify(body) }),
  assets: () => request<import("./types").Asset[]>("/api/assets"),
  createAsset: (body: unknown) => request("/api/assets", { method: "POST", headers: json, body: JSON.stringify(body) }),
  updateAsset: (id: string, body: unknown) => request(`/api/assets/${id}`, { method: "PUT", headers: json, body: JSON.stringify(body) }),
  deleteAsset: (id: string) => request(`/api/assets/${id}`, { method: "DELETE" }),
  createTx: (body: unknown) => request("/api/transactions", { method: "POST", headers: json, body: JSON.stringify(body) }),
  deleteTx: (id: string) => request(`/api/transactions/${id}`, { method: "DELETE" }),
  debts: () => request<import("./types").Debt[]>("/api/debts"),
  debt: (id: string) => request<import("./types").Debt>(`/api/debts/${id}`),
  createDebt: (body: unknown) => request("/api/debts", { method: "POST", headers: json, body: JSON.stringify(body) }),
  updateDebt: (id: string, body: unknown) => request(`/api/debts/${id}`, { method: "PUT", headers: json, body: JSON.stringify(body) }),
  deleteDebt: (id: string) => request(`/api/debts/${id}`, { method: "DELETE" }),
  payDebt: (id: string, body: unknown) => request(`/api/debts/${id}/pay`, { method: "POST", headers: json, body: JSON.stringify(body) }),
  payInstallment: (debtId: string, instId: string, body: unknown) =>
    request(`/api/debts/${debtId}/installments/${instId}/pay`, { method: "POST", headers: json, body: JSON.stringify(body) }),
  telegram: () => request<import("./types").TelegramStatus>("/api/telegram"),
  telegramTest: () => request("/api/telegram/test", { method: "POST" }),
  rates: () => request<import("./rates").Rates>("/api/rates"),
};
