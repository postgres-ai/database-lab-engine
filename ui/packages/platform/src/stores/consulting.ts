import { makeAutoObservable, runInAction } from "mobx";
import { request } from "../helpers/request";

const apiServer = process.env.REACT_APP_API_URL_PREFIX || '';

export interface Transaction {
  id: string;
  org_id: number;
  issue_id: number;
  amount: string;
  description?: string;
  source: string;
  created_at: string;
}

interface OrgBalance {
  org_id: number;
  balance: string;
}

class ConsultingStore {
  orgBalance: OrgBalance[] | null = null;
  transactions: Transaction[] = [];
  loading: boolean = false;
  error: string | null = null;

  constructor() {
    makeAutoObservable(this);
  }

  async getOrgBalance(orgId: number) {
    this.loading = true;
    this.error = null;

    try {
      const response = await request(`${apiServer}/org_balance?org_id=eq.${orgId}`, {
        method: "GET",
        headers: {

          Prefer: "return=representation",
        },
      });
      if (!response.ok) {
        console.error(`Error: ${response.statusText}`);
      }

      const data: OrgBalance[] = await response.json();
      runInAction(() => {
        this.orgBalance = data;
      });
    } catch (err: unknown) {
      runInAction(() => {
        if (err instanceof Error) {
          this.error = err.message || "Failed to fetch org_balance";
        } else {
          this.error = err as string;
        }
      });
    } finally {
      runInAction(() => {
        this.loading = false;
      });
    }
  }

  async getTransactions(orgId: number) {
    this.loading = true;
    this.error = null;

    try {
      const response = await request(`${apiServer}/consulting_transactions?org_id=eq.${orgId}`, {
        method: "GET",
        headers: {
          Prefer: "return=representation",
        },
      });
      if (!response.ok) {
        console.error(`Error: ${response.statusText}`);
      }

      const data: Transaction[] = await response.json();
      runInAction(() => {
        this.transactions = data;
      });
    } catch (err: unknown) {
      runInAction(() => {
        if (err instanceof Error) {
          this.error = err.message || "Failed to fetch transactions";
        } else {
          this.error = err as string;
        }
      });
    } finally {
      runInAction(() => {
        this.loading = false;
      });
    }
  }
}

export const consultingStore = new ConsultingStore();