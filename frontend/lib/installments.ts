export type PlannedInstallment = {
  index: number;
  due: string;
  amount: number;
  last: boolean;
};

export function todayISO() {
  const d = new Date();
  const z = new Date(d.getTime() - d.getTimezoneOffset() * 60000);
  return z.toISOString().slice(0, 10);
}

export function addMonths(iso: string, n: number): string {
  const [y, m, d] = iso.split("-").map(Number);
  if (!y || !m || !d) return iso;
  const base = Date.UTC(y, m - 1 + n, 1);
  const dt = new Date(base);
  const last = new Date(Date.UTC(dt.getUTCFullYear(), dt.getUTCMonth() + 1, 0)).getUTCDate();
  dt.setUTCDate(Math.min(d, last));
  return dt.toISOString().slice(0, 10);
}

export function autoInstallmentCount(total: number, monthly: number) {
  if (total <= 0) return 0;
  const per = monthly > 0 ? monthly : total;
  const pay = Math.min(per, total);
  if (pay <= 0) return 0;
  const full = Math.floor(total / pay);
  return total % pay === 0 ? full : full + 1;
}

export function splitAmounts(total: number, monthly: number, count = 0): number[] {
  if (total <= 0) return [];
  const per = monthly > 0 ? Math.min(monthly, total) : total;
  let n = count > 0 ? count : autoInstallmentCount(total, per);
  if (n < 1) n = 1;
  if (n > 360) n = 360;
  if ((n - 1) * per >= total) {
    n = autoInstallmentCount(total, per);
  }
  const last = total - per * (n - 1);
  if (last <= 0) return [];
  return [...Array(n - 1).fill(per), last];
}

export function planInstallments(total: number, monthly: number, start: string, count = 0): PlannedInstallment[] {
  const amounts = splitAmounts(total, monthly, count);
  if (!amounts.length || !start) return [];
  return amounts.map((amount, i) => ({
    index: i + 1,
    due: addMonths(start, i),
    amount,
    last: i === amounts.length - 1 && (amounts.length === 1 || amount !== amounts[0]),
  }));
}
