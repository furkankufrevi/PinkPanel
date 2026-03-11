import { useQuery } from "@tanstack/react-query";
import { getDetailedHealth } from "@/api/health";

export function useDetailedHealth() {
  return useQuery({
    queryKey: ["health", "detailed"],
    queryFn: getDetailedHealth,
    refetchInterval: 30000,
  });
}
