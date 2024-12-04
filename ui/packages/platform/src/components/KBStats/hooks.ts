import { useEffect, useState } from 'react'
import { request } from "../../helpers/request";

export type KBStats = {
  category: 'articles' | 'docs' | 'src' | 'mbox',
  domain: string,
  total_count: number,
  count: number,
  last_document_date: string
}

type UseKBStats = {
  data: KBStats[] | null,
  error: string | null,
  loading: boolean
}

export const useKBStats = (): UseKBStats => {
  const [data, setData] = useState<KBStats[] | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const apiServer = process.env.REACT_APP_API_URL_PREFIX || '';
  useEffect(() => {
    const fetchData = async () => {
      setLoading(true)
      try {
        const response = await request("/kb_category_domain_counts", {}, apiServer)
        const result: KBStats[] = await response.json();
        setData(result);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setLoading(false);
      }
    };

    fetchData().catch(console.error);
  }, []);

  return { data, loading, error };
};