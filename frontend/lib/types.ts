export type Account = {
  id: string;
  name: string;
  bank_name: string;
  account_number: string;
  type: string;
  balance: number;
  color: string;
  icon: string;
  created_at: string;
  updated_at: string;
};

export type ProjectItem = {
  id: string;
  project_id: string;
  name: string;
  planned_amount: number;
  paid_amount: number;
  notes: string;
  created_at: string;
  updated_at: string;
};

export type Asset = {
  id: string;
  name: string;
  type: string;
  quantity: number;
  unit: string;
  unit_value: number;
  value: number;
  notes: string;
  color: string;
  source_project_id?: string | null;
  source_project_name?: string;
  created_at: string;
  updated_at: string;
};

export type Project = {
  id: string;
  name: string;
  description: string;
  target_amount: number;
  current_amount: number;
  base_amount: number;
  deadline?: string | null;
  color: string;
  status: string;
  asset_id?: string | null;
  result_asset_type: string;
  items?: ProjectItem[];
  asset?: Asset | null;
  created_at: string;
  updated_at: string;
};

export type Tx = {
  id: string;
  account_id?: string | null;
  to_account_id?: string | null;
  project_id?: string | null;
  item_id?: string | null;
  type: string;
  amount: number;
  category: string;
  description: string;
  occurred_at: string;
  created_at: string;
  account_name?: string;
  to_account_name?: string;
  project_name?: string;
};

export type Installment = {
  id: string;
  debt_id: string;
  debt_name?: string;
  creditor?: string;
  amount: number;
  due_date: string;
  paid_at?: string | null;
  status: string;
  account_id?: string | null;
};

export type Debt = {
  id: string;
  name: string;
  type: string;
  creditor: string;
  total_amount: number;
  remaining: number;
  commission_amount?: number;
  total_cost?: number;
  net_received?: number;
  notes: string;
  color: string;
  has_schedule: boolean;
  start_date?: string | null;
  end_date?: string | null;
  monthly_amount: number;
  due_day: number;
  status: string;
  next_due?: string | null;
  installments?: Installment[];
  created_at: string;
  updated_at: string;
};

export type TelegramStatus = {
  telegram_chat_id: string;
  telegram_connected: boolean;
  bot_username: string;
};

export type AuthUser = {
  id: string;
  username: string;
  telegram_phone: string;
  telegram_verified: boolean;
  telegram_connected: boolean;
};

export type AuthStatus = {
  authenticated: boolean;
  mode: "register" | "login";
  has_users: boolean;
  bot_username: string;
  bot_ready: boolean;
  user?: AuthUser;
};

export type AuthChallenge = {
  id: string;
  status: "pending_start" | "waiting_contact" | "phone_mismatch" | "otp_sent" | "consumed" | string;
  purpose: string;
  bot_link?: string;
  bot_username?: string;
  phone_masked?: string;
  expires_in: number;
};

export type Dashboard = {
  net_worth: number;
  liquid: number;
  assets_total: number;
  project_spend: number;
  monthly_income: number;
  monthly_expense: number;
  account_count: number;
  project_count: number;
  accounts: Account[];
  projects: Project[];
  assets: Asset[];
  debts_remaining: number;
  upcoming: Installment[];
  recent: Tx[];
  cashflow: { date: string; income: number; expense: number }[];
  category_spend: { category: string; amount: number }[];
  monthly: { month: string; income: number; expense: number }[];
};
