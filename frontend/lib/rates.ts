export type Rates = {
  usd_toman: number;
  eur_toman: number;
  gold18_toman: number;
  gold24_toman: number;
  source: string;
  updated_at: string;
};

export function convertToman(toman: number, rates: Rates) {
  return {
    toman,
    usd: rates.usd_toman > 0 ? toman / rates.usd_toman : 0,
    eur: rates.eur_toman > 0 ? toman / rates.eur_toman : 0,
  };
}

export function liveUnitPrice(type: string, rates?: Rates | null): number {
  if (!rates) return 0;
  switch (type) {
    case "gold18":
    case "gold":
      return rates.gold18_toman;
    case "gold24":
      return rates.gold24_toman;
    case "usd":
      return rates.usd_toman;
    case "eur":
      return rates.eur_toman;
    default:
      return 0;
  }
}

export function liveAssetValue(type: string, qty: number, book: number, rates?: Rates | null): number {
  const unit = liveUnitPrice(type, rates);
  if (unit > 0 && qty > 0) return Math.round(qty * unit);
  return book;
}

export function isMarketAsset(type: string) {
  return type === "gold" || type === "gold18" || type === "gold24" || type === "usd" || type === "eur";
}

export function calcNetWorth(
  liquid: number,
  assets: { type: string; quantity: number; value: number }[],
  debts: number,
  rates?: Rates | null,
) {
  const assetsLive = assets.reduce((s, a) => s + liveAssetValue(a.type, a.quantity, a.value, rates), 0);
  return { assetsLive, netWorth: liquid + assetsLive - (debts || 0) };
}
