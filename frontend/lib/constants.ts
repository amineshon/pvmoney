export const MAX_MONEY = 10_000_000_000_000;

export const ACCOUNT_TYPES = [
  { id: "bank", unit: "" },
  { id: "cash", unit: "" },
  { id: "wallet", unit: "" },
  { id: "invest", unit: "" },
];

export const ACCOUNT_COLORS = ["#1d4e6e", "#0f3d3e", "#3b2a1a", "#2b2154", "#3a1d2e", "#1f3b2d", "#4a3210", "#243044"];

export const ASSET_TYPES = [
  { id: "gold18", unit: "gram" },
  { id: "gold24", unit: "gram" },
  { id: "gold", unit: "gram" },
  { id: "usd", unit: "dollar" },
  { id: "eur", unit: "euro" },
  { id: "btc", unit: "btc" },
  { id: "eth", unit: "eth" },
  { id: "house", unit: "property" },
  { id: "land", unit: "meter" },
  { id: "car", unit: "car" },
  { id: "stock", unit: "share" },
  { id: "other", unit: "piece" },
];

export const DEBT_TYPES = ["loan", "personal", "credit"];
export const DEBT_COLORS = ["#fb7185", "#f97316", "#e11d48", "#a21caf", "#7c3aed", "#0ea5e9"];

export const INCOME_CATS = ["salary", "freelance", "investment", "gift", "sale", "other_income"];
export const EXPENSE_CATS = ["food", "transport", "housing", "bills", "shopping", "health", "fun", "education", "other_expense"];

export function typeLabel(t: string): string {
  return t;
}
