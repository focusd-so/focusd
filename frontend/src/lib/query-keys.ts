// Centralised React Query keys for backend-derived state. Keeping them in one
// file avoids stringly-typed drift between the call sites and the event bridge
// that invalidates them.

export const queryKeys = {
  protectionStatus: ["protection", "status"] as const,
  pauseHistory: (days: number) => ["protection", "history", days] as const,
  pauseHistoryAll: ["protection", "history"] as const,

  allowList: ["allow", "list"] as const,

  recentUsages: ["usage", "recent"] as const,
  usageList: (filters: unknown) => ["usage", "list", filters] as const,
  usageListAll: ["usage", "list"] as const,
  usageAggregation: (filters: unknown) => ["usage", "aggregation", filters] as const,
  usageAggregationAll: ["usage", "aggregation"] as const,

  applicationList: ["application", "list"] as const,
  dayInsights: (date: string) => ["insights", "day", date] as const,
  dayInsightsAll: ["insights", "day"] as const,
  sandboxLogs: (logType: string, search: string) => ["sandbox", "logs", logType, search] as const,
  sandboxLogsAll: ["sandbox", "logs"] as const,
} as const;
