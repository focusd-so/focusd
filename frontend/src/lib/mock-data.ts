// Centralized mock data for all Screen Time views

export interface DailyStats {
  date: number; // unix timestamp (start of day)
  productiveMinutes: number;
  neutralMinutes: number;
  distractingMinutes: number;
  focusScore: number;
  deepWorkSessions: number;
  longestSessionMinutes: number;
  blockedAttempts: number;
  contextSwitches: number;
}

export interface HourlyStats {
  hour: number; // 0-23
  productiveMinutes: number;
  distractingMinutes: number;
  neutralMinutes: number;
}

export interface ProjectStats {
  id: string;
  name: string;
  totalMinutes: number;
  sessionsCount: number;
  lastActive: number;
  classification: "productive" | "mixed";
}

export interface DeepWorkSession {
  id: string;
  projectName: string;
  startTime: number;
  durationMinutes: number;
  app: string;
  appIcon?: string;
}

export interface BlockedAttempt {
  id: string;
  hostname: string;
  appName: string;
  icon?: string;
  count: number;
  lastAttempt: number;
  peakHour: number;
  tags: string[];
}

export interface AppUsageStats {
  id: string;
  name: string;
  appId: string;
  hostname?: string;
  icon?: string;
  totalMinutes: number;
  sessionsCount: number;
  classification: "productive" | "distracting" | "neutral";
}

export interface DayData {
  stats: DailyStats;
  hourlyBreakdown: HourlyStats[];
  projects: ProjectStats[];
  deepWorkSessions: DeepWorkSession[];
  blockedAttempts: BlockedAttempt[];
  aiSummary: AIDailySummary;
  communicationChannels: CommunicationChannel[];
  topDistractions: DistractionItem[];
}

// AI-generated daily summary
export interface AIDailySummary {
  tldr: string;
  peakFocusWindow: string;
  dangerZone: string;
  topApp: { name: string; minutes: number };
  blockedCount: number;
  tip: string;
  fullMarkdown: string;
}

// Communication channel breakdown
export interface CommunicationChannel {
  id: string;
  name: string;
  icon: string;
  minutes: number;
  sessions: number;
}

// Top distracting apps/sites
export interface DistractionItem {
  id: string;
  name: string;
  hostname?: string;
  icon?: string;
  minutes: number;
  category: string;
}

// Get start of day timestamp
function getStartOfDay(date: Date): number {
  const d = new Date(date);
  d.setHours(0, 0, 0, 0);
  return Math.floor(d.getTime() / 1000);
}

// Generate dates for last N days
function getLastNDays(n: number): Date[] {
  const dates: Date[] = [];
  const now = new Date();
  for (let i = 0; i < n; i++) {
    const date = new Date(now);
    date.setDate(date.getDate() - i);
    date.setHours(0, 0, 0, 0);
    dates.push(date);
  }
  return dates;
}

// Generate hourly breakdown for a day
function generateHourlyBreakdown(seed: number): HourlyStats[] {
  const hours: HourlyStats[] = [];
  for (let hour = 0; hour < 24; hour++) {
    // Work hours (9-18) have more activity
    const isWorkHour = hour >= 9 && hour <= 18;
    const baseProductive = isWorkHour ? 30 + (seed % 25) : seed % 5;
    const baseDistracting = isWorkHour ? 2 + (seed % 8) : seed % 3;
    const baseNeutral = isWorkHour ? 5 + (seed % 10) : seed % 2;

    // Add some variation based on hour
    const peakBonus = (hour >= 10 && hour <= 12) || (hour >= 14 && hour <= 16) ? 10 : 0;
    const lunchDip = hour === 12 || hour === 13 ? -15 : 0;

    hours.push({
      hour,
      productiveMinutes: Math.max(0, baseProductive + peakBonus + lunchDip + ((seed * hour) % 10)),
      distractingMinutes: Math.max(0, baseDistracting + ((seed * hour) % 5)),
      neutralMinutes: Math.max(0, baseNeutral + ((seed * hour) % 5)),
    });
  }
  return hours;
}

// Generate projects for a day
function generateProjects(seed: number, dayOffset: number): ProjectStats[] {
  const projectNames = [
    "focusd",
    "api-gateway",
    "auth-service",
    "docs-site",
    "mobile-app",
    "dashboard",
  ];

  const numProjects = 3 + (seed % 3);
  const projects: ProjectStats[] = [];

  for (let i = 0; i < numProjects; i++) {
    const name = projectNames[(seed + i) % projectNames.length];
    const baseMinutes = 180 - i * 40 + ((seed * (i + 1)) % 60);
    projects.push({
      id: `${dayOffset}-${i}`,
      name,
      totalMinutes: Math.max(20, baseMinutes),
      sessionsCount: 2 + ((seed + i) % 6),
      lastActive: Date.now() / 1000 - dayOffset * 86400 - i * 3600,
      classification: i < 2 ? "productive" : "mixed",
    });
  }

  return projects.sort((a, b) => b.totalMinutes - a.totalMinutes);
}

// Generate deep work sessions for a day
function generateDeepWorkSessions(
  seed: number,
  dayOffset: number,
  projects: ProjectStats[]
): DeepWorkSession[] {
  const numSessions = 1 + (seed % 4);
  const sessions: DeepWorkSession[] = [];

  for (let i = 0; i < numSessions; i++) {
    const project = projects[i % projects.length];
    const duration = 25 + ((seed * (i + 1)) % 80);
    sessions.push({
      id: `${dayOffset}-session-${i}`,
      projectName: project.name,
      startTime:
        Date.now() / 1000 -
        dayOffset * 86400 -
        (10 + i * 3) * 3600 +
        (seed % 1800),
      durationMinutes: duration,
      app: "Visual Studio Code",
    });
  }

  return sessions.sort((a, b) => b.startTime - a.startTime);
}

// Generate blocked attempts for a day
function generateBlockedAttempts(seed: number, dayOffset: number): BlockedAttempt[] {
  const sites = [
    { hostname: "twitter.com", tags: ["social_media"] },
    { hostname: "reddit.com", tags: ["social_media", "content_consumption"] },
    { hostname: "youtube.com", tags: ["video", "streaming"] },
    { hostname: "instagram.com", tags: ["social_media"] },
    { hostname: "facebook.com", tags: ["social_media"] },
    { hostname: "news.ycombinator.com", tags: ["news", "tech"] },
  ];

  const numBlocked = 1 + (seed % 4);
  const attempts: BlockedAttempt[] = [];

  for (let i = 0; i < numBlocked; i++) {
    const site = sites[(seed + i) % sites.length];
    const count = 2 + ((seed * (i + 1)) % 10);
    attempts.push({
      id: `${dayOffset}-blocked-${i}`,
      hostname: site.hostname,
      appName: "Google Chrome",
      count,
      lastAttempt: Date.now() / 1000 - dayOffset * 86400 - i * 7200,
      peakHour: 14 + (i % 4),
      tags: site.tags,
    });
  }

  return attempts.sort((a, b) => b.count - a.count);
}

// Generate AI daily summary based on day stats
function generateAISummary(
  stats: DailyStats,
  hourlyBreakdown: HourlyStats[],
  blockedAttempts: BlockedAttempt[]
): AIDailySummary {
  // Find peak focus window (hour with most productive minutes)
  const peakHour = hourlyBreakdown.reduce((best, h) =>
    h.productiveMinutes > best.productiveMinutes ? h : best
  );
  const peakWindow = `${peakHour.hour}:00 - ${peakHour.hour + 1}:00`;

  // Find danger zone (hour with most distracting minutes during work hours)
  const workHours = hourlyBreakdown.filter((h) => h.hour >= 9 && h.hour <= 18);
  const dangerHour = workHours.reduce((worst, h) =>
    h.distractingMinutes > worst.distractingMinutes ? h : worst
  );
  const dangerZone = `${dangerHour.hour}:00 - ${dangerHour.hour + 1}:00`;

  const tips = [
    "Try time-blocking your most important tasks during your peak hours.",
    "Consider a 20-min walk after lunch to reset your focus.",
    "Your morning momentum is strong - protect it from meetings.",
    "Batch your communication checks to reduce context switches.",
    "The post-lunch dip is real. Schedule lighter tasks for 2-3 PM.",
  ];

  const totalBlocked = blockedAttempts.reduce((sum, b) => sum + b.count, 0);
  const topBlockedSite = blockedAttempts[0]?.hostname || "none";

  const tldr =
    stats.focusScore >= 75
      ? `Great focus day! You maintained ${stats.focusScore}% productivity and blocked ${totalBlocked} distractions.`
      : stats.focusScore >= 50
        ? `Decent focus with room for improvement. ${stats.focusScore}% productive, with ${stats.distractingMinutes}m lost to distractions.`
        : `Challenging focus day at ${stats.focusScore}%. Tomorrow, try protecting your peak hours from interruptions.`;

  const fullMarkdown = `## Daily Focus Report

### Quick Summary
${tldr}

### Peak Performance
- **Best focus window:** ${peakWindow}
- **Top productive app:** VS Code (${formatMinutes(stats.productiveMinutes * 0.6)})
- **Deep work sessions:** ${stats.deepWorkSessions}

### Watch Out
- **Danger zone:** ${dangerZone} (most distractions)
- **Top distraction source:** ${topBlockedSite}
- **Context switches:** ${stats.contextSwitches}

### Blocked Attempts
You successfully blocked **${totalBlocked}** attempts to access distracting sites.

### Tip of the Day
${tips[stats.date % tips.length]}`;

  return {
    tldr,
    peakFocusWindow: peakWindow,
    dangerZone,
    topApp: { name: "VS Code", minutes: Math.round(stats.productiveMinutes * 0.6) },
    blockedCount: totalBlocked,
    tip: tips[stats.date % tips.length],
    fullMarkdown,
  };
}

// Generate communication channel breakdown
function generateCommunicationBreakdown(seed: number): CommunicationChannel[] {
  const channels = [
    { name: "Slack", icon: "💬", baseMinutes: 80 },
    { name: "Email", icon: "📧", baseMinutes: 45 },
    { name: "Zoom", icon: "📹", baseMinutes: 30 },
    { name: "Discord", icon: "🎮", baseMinutes: 15 },
    { name: "Teams", icon: "📋", baseMinutes: 10 },
  ];

  return channels.map((channel, i) => ({
    id: `comm-${i}`,
    name: channel.name,
    icon: channel.icon,
    minutes: Math.max(0, channel.baseMinutes + ((seed * (i + 1)) % 30) - 10),
    sessions: 5 + ((seed + i) % 20),
  }));
}

// Generate top distracting apps/sites
function generateTopDistractions(seed: number): DistractionItem[] {
  const distractions = [
    { name: "YouTube", hostname: "youtube.com", category: "Video", baseMinutes: 45 },
    { name: "Reddit", hostname: "reddit.com", category: "Social", baseMinutes: 32 },
    { name: "Twitter", hostname: "twitter.com", category: "Social", baseMinutes: 18 },
    { name: "Instagram", hostname: "instagram.com", category: "Social", baseMinutes: 12 },
    { name: "TikTok", hostname: "tiktok.com", category: "Video", baseMinutes: 8 },
  ];

  return distractions.map((d, i) => ({
    id: `distraction-${i}`,
    name: d.name,
    hostname: d.hostname,
    category: d.category,
    minutes: Math.max(2, d.baseMinutes + ((seed * (i + 1)) % 15) - 5),
  })).sort((a, b) => b.minutes - a.minutes);
}

// Generate complete data for a specific day
function generateDayData(date: Date, dayOffset: number): DayData {
  const seed = date.getDate() * 7 + date.getMonth() * 31;
  const hourly = generateHourlyBreakdown(seed);

  const productiveMinutes = hourly.reduce((sum, h) => sum + h.productiveMinutes, 0);
  const distractingMinutes = hourly.reduce((sum, h) => sum + h.distractingMinutes, 0);
  const neutralMinutes = hourly.reduce((sum, h) => sum + h.neutralMinutes, 0);
  const totalActive = productiveMinutes + distractingMinutes;

  const projects = generateProjects(seed, dayOffset);
  const deepWorkSessions = generateDeepWorkSessions(seed, dayOffset, projects);
  const blockedAttempts = generateBlockedAttempts(seed, dayOffset);
  const communicationChannels = generateCommunicationBreakdown(seed);
  const topDistractions = generateTopDistractions(seed);

  const stats: DailyStats = {
    date: getStartOfDay(date),
    productiveMinutes,
    neutralMinutes,
    distractingMinutes,
    focusScore: totalActive > 0 ? Math.round((productiveMinutes / totalActive) * 100) : 0,
    deepWorkSessions: deepWorkSessions.length,
    longestSessionMinutes: Math.max(...deepWorkSessions.map((s) => s.durationMinutes), 0),
    blockedAttempts: blockedAttempts.reduce((sum, b) => sum + b.count, 0),
    contextSwitches: 8 + (seed % 15),
  };

  const aiSummary = generateAISummary(stats, hourly, blockedAttempts);

  return {
    stats,
    hourlyBreakdown: hourly,
    projects,
    deepWorkSessions,
    blockedAttempts,
    aiSummary,
    communicationChannels,
    topDistractions,
  };
}

// Pre-generate data for last 30 days
const daysData = new Map<number, DayData>();
const dates = getLastNDays(30);
dates.forEach((date, i) => {
  const data = generateDayData(date, i);
  daysData.set(data.stats.date, data);
});

// Public API: Get data for a specific date
export function getDataForDate(date: Date): DayData {
  const startOfDay = getStartOfDay(date);
  const existing = daysData.get(startOfDay);
  if (existing) return existing;

  // Generate if not found (future or very old dates)
  const dayOffset = Math.floor((Date.now() / 1000 - startOfDay) / 86400);
  return generateDayData(date, dayOffset);
}

// Public API: Get weekly stats for trends page
export function getWeeklyStats(): DailyStats[] {
  const last7Days = getLastNDays(7);
  return last7Days.map((date) => getDataForDate(date).stats).reverse();
}

// Legacy exports for backward compatibility
export const mockWeeklyStats = getWeeklyStats();
export const mockTodayStats = getDataForDate(new Date()).stats;
export const mockProjects = getDataForDate(new Date()).projects;
export const mockDeepWorkSessions = getDataForDate(new Date()).deepWorkSessions;
export const mockBlockedAttempts = getDataForDate(new Date()).blockedAttempts;

// App usage (not date-specific for now)
export const mockAppUsage: AppUsageStats[] = [
  {
    id: "1",
    name: "Visual Studio Code",
    appId: "com.microsoft.VSCode",
    totalMinutes: 245,
    sessionsCount: 12,
    classification: "productive",
  },
  {
    id: "2",
    name: "Google Chrome",
    appId: "com.google.Chrome",
    hostname: "github.com",
    totalMinutes: 89,
    sessionsCount: 25,
    classification: "productive",
  },
  {
    id: "3",
    name: "Slack",
    appId: "com.tinyspeck.slackmacgap",
    totalMinutes: 67,
    sessionsCount: 45,
    classification: "neutral",
  },
  {
    id: "4",
    name: "Terminal",
    appId: "com.apple.Terminal",
    totalMinutes: 43,
    sessionsCount: 18,
    classification: "productive",
  },
  {
    id: "5",
    name: "Google Chrome",
    appId: "com.google.Chrome",
    hostname: "stackoverflow.com",
    totalMinutes: 28,
    sessionsCount: 8,
    classification: "productive",
  },
];

// Helper to format minutes as "Xh Ym"
export function formatMinutes(minutes: number): string {
  return formatDuration(minutes * 60);
}

// Helper to format duration in a human readable way
export function formatDuration(seconds: number): string {
  if (seconds < 60) {
    return `${Math.round(seconds)}s`;
  }
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);

  if (h > 0) {
    if (m > 0) return `${h}h ${m}m`;
    return `${h}h`;
  }
  return `${m}m`;
}

// Helper to format relative time
export function formatRelativeTime(unixSeconds: number): string {
  const now = Date.now() / 1000;
  const diff = now - unixSeconds;

  if (diff < 60) return "just now";
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return `${Math.floor(diff / 86400)}d ago`;
}

// Helper to format date for display
export function formatDate(date: Date): string {
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const yesterday = new Date(today);
  yesterday.setDate(yesterday.getDate() - 1);
  const targetDay = new Date(date.getFullYear(), date.getMonth(), date.getDate());

  if (targetDay.getTime() === today.getTime()) {
    return "Today";
  }
  if (targetDay.getTime() === yesterday.getTime()) {
    return "Yesterday";
  }

  return date.toLocaleDateString("en-US", {
    weekday: "long",
    month: "long",
    day: "numeric",
    year: date.getFullYear() !== now.getFullYear() ? "numeric" : undefined,
  });
}

// Helper to check if date is today
export function isToday(date: Date): boolean {
  const now = new Date();
  return (
    date.getDate() === now.getDate() &&
    date.getMonth() === now.getMonth() &&
    date.getFullYear() === now.getFullYear()
  );
}

// ---------------------------------------------------------------------------
// Backend-shaped fixtures (timeline events, DayInsights, etc).
// Used as dev fallbacks while the timeline rewrite of GetUsageList /
// GetUsageAggregation / GetDayInsights / GetSandboxExecutionLogs is pending.
// ---------------------------------------------------------------------------

import { Event as TimelineEvent } from "../../bindings/github.com/focusd-so/focusd/internal/timeline/models";
import {
  ApplicationUsagePayload,
  ClassificationResult,
  CommunicationBreakdown,
  CustomRulesTracePayload,
  DayInsights,
  EnforcementResult,
  LLMDailySummary,
  ProductivityScore,
} from "../../bindings/github.com/focusd-so/focusd/internal/usage/models";

let mockEventIdCounter = 1_000_000;
function nextMockEventId(): number {
  mockEventIdCounter += 1;
  return mockEventIdCounter;
}

interface MockUsageSeed {
  applicationId: number;
  appName: string;
  hostname?: string;
  windowTitle: string;
  classification: "productive" | "distracting" | "neutral";
  classificationSource: "obviously" | "custom_rules" | "llm_openai";
  classificationReason?: string;
  enforcementAction: "allow" | "block" | "paused";
  enforcementSource?: "application" | "custom_rules" | "allowed" | "paused";
  minutesAgo: number;
  durationSeconds?: number;
  tags?: string[];
}

function buildUsagePayload(seed: MockUsageSeed): ApplicationUsagePayload {
  const enforcement = new EnforcementResult({
    StandardEnforcementResult: {
      Action: seed.enforcementAction,
      Reason:
        seed.enforcementAction === "block"
          ? "distracting usage, focus protection is enabled"
          : seed.enforcementAction === "paused"
            ? "focus protection is temporarily paused by user"
            : "non distracting usage",
      Source: seed.enforcementSource ?? "application",
    },
    CustomRulesEnforcementResult: null,
  });

  const classification = new ClassificationResult({
    custom_rules_classification_result: null,
    llm_classification_result: null,
    obviously_classification_result: null,
  });

  return new ApplicationUsagePayload({
    application_id: seed.applicationId,
    window_title: seed.windowTitle,
    browser_url: seed.hostname ? `https://${seed.hostname}/` : undefined,
    classification: seed.classification,
    classification_reason: seed.classificationReason ?? "",
    classification_source: seed.classificationSource,
    classification_result: classification,
    enforcement_result: enforcement,
    tags: seed.tags ?? [],
  });
}

function buildUsageEvent(seed: MockUsageSeed): TimelineEvent {
  const occurredAt = Math.floor(Date.now() / 1000) - seed.minutesAgo * 60;
  const finishedAt = seed.durationSeconds
    ? occurredAt + seed.durationSeconds
    : null;

  const event = new TimelineEvent({
    id: nextMockEventId(),
    occurred_at: occurredAt,
    type: "usage_changed",
    payload: JSON.stringify(buildUsagePayload(seed)),
    trace_id: "",
    parent_id: null,
    ended_at: finishedAt,
    key: null,
    tags: [],
  });
  return event;
}

const mockUsageSeeds: MockUsageSeed[] = [
  {
    applicationId: 1,
    appName: "Visual Studio Code",
    windowTitle: "focusd — main.go",
    classification: "productive",
    classificationSource: "obviously",
    enforcementAction: "allow",
    minutesAgo: 1,
    durationSeconds: 25 * 60,
    tags: ["coding"],
  },
  {
    applicationId: 2,
    appName: "Google Chrome",
    hostname: "twitter.com",
    windowTitle: "Home / X",
    classification: "distracting",
    classificationSource: "obviously",
    enforcementAction: "block",
    enforcementSource: "application",
    minutesAgo: 6,
    durationSeconds: 45,
    tags: ["social_media"],
  },
  {
    applicationId: 3,
    appName: "Slack",
    windowTitle: "Acme · #engineering",
    classification: "neutral",
    classificationSource: "obviously",
    enforcementAction: "allow",
    minutesAgo: 12,
    durationSeconds: 5 * 60,
    tags: ["communication"],
  },
  {
    applicationId: 4,
    appName: "Google Chrome",
    hostname: "youtube.com",
    windowTitle: "How browsers work — YouTube",
    classification: "distracting",
    classificationSource: "custom_rules",
    classificationReason: "matched custom rule: video sites during work hours",
    enforcementAction: "block",
    enforcementSource: "custom_rules",
    minutesAgo: 25,
    durationSeconds: 30,
    tags: ["video"],
  },
  {
    applicationId: 5,
    appName: "Visual Studio Code",
    windowTitle: "focusd — protection.go",
    classification: "productive",
    classificationSource: "obviously",
    enforcementAction: "allow",
    minutesAgo: 60,
    durationSeconds: 35 * 60,
    tags: ["coding"],
  },
];

export function mockRecentUsageEvents(): TimelineEvent[] {
  return mockUsageSeeds.map(buildUsageEvent);
}

export interface MockUsageAggregationItem {
  application: {
    id: number;
    name: string;
    icon: string | null;
    hostname: string | null;
    domain: string | null;
  };
  total_duration: number;
  usage_count: number;
}

export function mockUsageAggregation(): MockUsageAggregationItem[] {
  return mockAppUsage.map((app, i) => ({
    application: {
      id: i + 1,
      name: app.name,
      icon: null,
      hostname: app.hostname ?? null,
      domain: app.hostname ?? null,
    },
    total_duration: app.totalMinutes * 60,
    usage_count: app.sessionsCount,
  }));
}

export function mockDayInsights(): DayInsights {
  const today = getDataForDate(new Date());

  const productivityScore = new ProductivityScore({
    productive_seconds: today.stats.productiveMinutes * 60,
    distracting_seconds: today.stats.distractingMinutes * 60,
    idle_seconds: today.stats.neutralMinutes * 60,
    other_seconds: 0,
    productivity_score: today.stats.focusScore,
  });

  const hourly: Record<string, ProductivityScore> = {};
  for (const slot of today.hourlyBreakdown) {
    hourly[String(slot.hour)] = new ProductivityScore({
      productive_seconds: slot.productiveMinutes * 60,
      distracting_seconds: slot.distractingMinutes * 60,
      idle_seconds: slot.neutralMinutes * 60,
      other_seconds: 0,
      productivity_score:
        slot.productiveMinutes + slot.distractingMinutes > 0
          ? Math.round(
              (slot.productiveMinutes /
                (slot.productiveMinutes + slot.distractingMinutes)) *
                100,
            )
          : 0,
    });
  }

  const topDistractions: Record<string, number> = {};
  for (const item of today.topDistractions) {
    topDistractions[item.hostname ?? item.name] = item.minutes * 60;
  }

  const topBlocked: Record<string, number> = {};
  for (const blocked of today.blockedAttempts) {
    topBlocked[blocked.hostname] = blocked.count * 60;
  }

  const projectBreakdown: Record<string, number> = {};
  for (const project of today.projects) {
    projectBreakdown[project.name] = project.totalMinutes * 60;
  }

  const communicationBreakdown: Record<string, CommunicationBreakdown> = {};
  for (const channel of today.communicationChannels) {
    communicationBreakdown[channel.name] = new CommunicationBreakdown({
      name: channel.name,
      channel: channel.name,
      duration_seconds: channel.minutes * 60,
    });
  }

  const summary = new LLMDailySummary({
    id: 1,
    date: new Date().toISOString().slice(0, 10),
    headline: today.aiSummary.tldr.slice(0, 60),
    narrative: today.aiSummary.tldr,
    key_pattern: today.aiSummary.dangerZone,
    wins: JSON.stringify([
      `Peak focus window ${today.aiSummary.peakFocusWindow}`,
      `Top app: ${today.aiSummary.topApp.name} ${formatMinutes(today.aiSummary.topApp.minutes)}`,
    ]),
    suggestion: today.aiSummary.tip,
    day_vibe: today.stats.focusScore >= 75 ? "locked-in" : "balanced",
    context_switch_count: today.stats.contextSwitches,
    longest_focus_minutes: today.stats.longestSessionMinutes,
    deep_work_minutes: today.stats.deepWorkSessions * 30,
    blocked_attempt_count: today.stats.blockedAttempts,
    created_at: Math.floor(Date.now() / 1000),
  });

  return new DayInsights({
    productivity_score: productivityScore,
    productivity_per_hour_breakdown: hourly,
    llm_daily_summary: summary,
    top_distractions: topDistractions,
    top_blocked: topBlocked,
    project_breakdown: projectBreakdown,
    communication_breakdown: communicationBreakdown,
  });
}

export function mockSandboxLogEvents(logType: string, search: string): TimelineEvent[] {
  const samples: Array<{ context: object; output: string; logs?: string[]; error?: string }> = [
    {
      context: {
        usage: {
          app: "Slack",
          host: undefined,
          title: "#engineering · Acme",
        },
      },
      output: `{"classification":"productive","classification_reason":"native app rule matched"}`,
      logs: ["[rule] matched native_app:Slack", "[rule] returning productive"],
    },
    {
      context: {
        usage: {
          app: "Google Chrome",
          host: "youtube.com",
          title: "Home — YouTube",
        },
      },
      output: `{"enforcementAction":"block","enforcementReason":"video sites during work hours"}`,
      logs: ["[rule] hostname=youtube.com matched", "[rule] working hours -> block"],
    },
    {
      context: {
        usage: { app: "Visual Studio Code", title: "focusd — service.go" },
      },
      output: "",
      logs: ["[rule] no rule matched, deferring"],
      error: "",
    },
  ];

  const events = samples
    .filter((_sample, i) => {
      if (logType && i === 0 && logType !== "classification") return false;
      if (logType && i === 1 && logType !== "enforcement_action") return false;
      return true;
    })
    .map((sample, i) => {
      const occurredAt = Math.floor(Date.now() / 1000) - i * 90;
      const payload = new CustomRulesTracePayload({
        context: JSON.stringify(sample.context),
        logs: sample.logs ?? [],
        resp: sample.output,
        error: sample.error ?? "",
      });
      return new TimelineEvent({
        id: nextMockEventId(),
        occurred_at: occurredAt,
        type: "custom_rules_trace",
        payload: JSON.stringify(payload),
        trace_id: "",
        parent_id: null,
        ended_at: occurredAt,
        key: null,
        tags: [],
      });
    });

  if (!search) return events;
  const needle = search.toLowerCase();
  return events.filter((event) => event.payload.toLowerCase().includes(needle));
}
