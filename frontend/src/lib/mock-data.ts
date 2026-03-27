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
  bundleId: string;
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
    bundleId: "com.microsoft.VSCode",
    totalMinutes: 245,
    sessionsCount: 12,
    classification: "productive",
  },
  {
    id: "2",
    name: "Google Chrome",
    bundleId: "com.google.Chrome",
    hostname: "github.com",
    totalMinutes: 89,
    sessionsCount: 25,
    classification: "productive",
  },
  {
    id: "3",
    name: "Slack",
    bundleId: "com.tinyspeck.slackmacgap",
    totalMinutes: 67,
    sessionsCount: 45,
    classification: "neutral",
  },
  {
    id: "4",
    name: "Terminal",
    bundleId: "com.apple.Terminal",
    totalMinutes: 43,
    sessionsCount: 18,
    classification: "productive",
  },
  {
    id: "5",
    name: "Google Chrome",
    bundleId: "com.google.Chrome",
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
