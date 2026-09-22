async function parse<T>(res: Response): Promise<T> {
  if (res.status === 204) return undefined as T;
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error((data as { error?: string }).error || "server");
  }
  return data as T;
}

const json = { "Content-Type": "application/json" };

export const api = {
  dashboard: () => fetch("/api/dashboard").then((r) => parse<import("./types").Dashboard>(r)),
  accounts: () => fetch("/api/accounts").then((r) => parse<import("./types").Account[]>(r)),
  projects: () => fetch("/api/projects").then((r) => parse<import("./types").Project[]>(r)),
  project: (id: string) => fetch(`/api/projects/${id}`).then((r) => parse<import("./types").Project>(r)),
  transactions: (q = "") => fetch(`/api/transactions${q}`).then((r) => parse<import("./types").Tx[]>(r)),
  createAccount: (body: unknown) =>
    fetch("/api/accounts", { method: "POST", headers: json, body: JSON.stringify(body) }).then((r) => parse(r)),
  updateAccount: (id: string, body: unknown) =>
    fetch(`/api/accounts/${id}`, { method: "PUT", headers: json, body: JSON.stringify(body) }).then((r) => parse(r)),
  deleteAccount: (id: string) => fetch(`/api/accounts/${id}`, { method: "DELETE" }).then((r) => parse(r)),
  adjustAccount: (id: string, body: unknown) =>
    fetch(`/api/accounts/${id}/adjust`, { method: "POST", headers: json, body: JSON.stringify(body) }).then((r) => parse(r)),
  createProject: (body: unknown) =>
    fetch("/api/projects", { method: "POST", headers: json, body: JSON.stringify(body) }).then((r) => parse(r)),
  updateProject: (id: string, body: unknown) =>
    fetch(`/api/projects/${id}`, { method: "PUT", headers: json, body: JSON.stringify(body) }).then((r) => parse(r)),
  deleteProject: (id: string) => fetch(`/api/projects/${id}`, { method: "DELETE" }).then((r) => parse(r)),
  contribute: (id: string, body: unknown) =>
    fetch(`/api/projects/${id}/contribute`, { method: "POST", headers: json, body: JSON.stringify(body) }).then((r) => parse(r)),
  withdraw: (id: string, body: unknown) =>
    fetch(`/api/projects/${id}/withdraw`, { method: "POST", headers: json, body: JSON.stringify(body) }).then((r) => parse(r)),
  createItem: (projectId: string, body: unknown) =>
    fetch(`/api/projects/${projectId}/items`, { method: "POST", headers: json, body: JSON.stringify(body) }).then((r) => parse(r)),
  updateItem: (projectId: string, itemId: string, body: unknown) =>
    fetch(`/api/projects/${projectId}/items/${itemId}`, { method: "PUT", headers: json, body: JSON.stringify(body) }).then((r) => parse(r)),
  deleteItem: (projectId: string, itemId: string) => fetch(`/api/projects/${projectId}/items/${itemId}`, { method: "DELETE" }).then((r) => parse(r)),
  payItem: (projectId: string, itemId: string, body: unknown) =>
    fetch(`/api/projects/${projectId}/items/${itemId}/pay`, { method: "POST", headers: json, body: JSON.stringify(body) }).then((r) => parse(r)),
  convertAsset: (projectId: string, body: unknown) =>
    fetch(`/api/projects/${projectId}/convert-asset`, { method: "POST", headers: json, body: JSON.stringify(body) }).then((r) => parse(r)),
  assets: () => fetch("/api/assets").then((r) => parse<import("./types").Asset[]>(r)),
  createAsset: (body: unknown) =>
    fetch("/api/assets", { method: "POST", headers: json, body: JSON.stringify(body) }).then((r) => parse(r)),
  updateAsset: (id: string, body: unknown) =>
    fetch(`/api/assets/${id}`, { method: "PUT", headers: json, body: JSON.stringify(body) }).then((r) => parse(r)),
  deleteAsset: (id: string) => fetch(`/api/assets/${id}`, { method: "DELETE" }).then((r) => parse(r)),
  createTx: (body: unknown) =>
    fetch("/api/transactions", { method: "POST", headers: json, body: JSON.stringify(body) }).then((r) => parse(r)),
  deleteTx: (id: string) => fetch(`/api/transactions/${id}`, { method: "DELETE" }).then((r) => parse(r)),
  debts: () => fetch("/api/debts").then((r) => parse<import("./types").Debt[]>(r)),
  debt: (id: string) => fetch(`/api/debts/${id}`).then((r) => parse<import("./types").Debt>(r)),
  createDebt: (body: unknown) =>
    fetch("/api/debts", { method: "POST", headers: json, body: JSON.stringify(body) }).then((r) => parse(r)),
  updateDebt: (id: string, body: unknown) =>
    fetch(`/api/debts/${id}`, { method: "PUT", headers: json, body: JSON.stringify(body) }).then((r) => parse(r)),
  deleteDebt: (id: string) => fetch(`/api/debts/${id}`, { method: "DELETE" }).then((r) => parse(r)),
  payDebt: (id: string, body: unknown) =>
    fetch(`/api/debts/${id}/pay`, { method: "POST", headers: json, body: JSON.stringify(body) }).then((r) => parse(r)),
  payInstallment: (debtId: string, instId: string, body: unknown) =>
    fetch(`/api/debts/${debtId}/installments/${instId}/pay`, { method: "POST", headers: json, body: JSON.stringify(body) }).then((r) => parse(r)),
  telegram: () => fetch("/api/telegram").then((r) => parse<import("./types").TelegramStatus>(r)),
  telegramTest: () => fetch("/api/telegram/test", { method: "POST" }).then((r) => parse(r)),
  rates: () => fetch("/api/rates").then((r) => parse<import("./rates").Rates>(r)),
};
