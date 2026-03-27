export const RUNTIME_TYPES_FILE_PATH = "file:///focusd-runtime.d.ts";

export const RUNTIME_TYPES_SOURCE = `declare module "@focusd/runtime" {
  export type ClassificationType = "unknown" | "productive" | "distracting" | "neutral" | "system";
  export type EnforcementActionType = "none" | "block" | "paused" | "allow";
  export type WeekdayType = "Sunday" | "Monday" | "Tuesday" | "Wednesday" | "Thursday" | "Friday" | "Saturday";
  export type Minutes = number;

  export const Classification: {
    readonly Unknown: "unknown";
    readonly Productive: "productive";
    readonly Distracting: "distracting";
    readonly Neutral: "neutral";
    readonly System: "system";
  };

  export const EnforcementAction: {
    readonly None: "none";
    readonly Block: "block";
    readonly Paused: "paused";
    readonly Allow: "allow";
  };

  export const Timezone: {
    readonly America_New_York: "America/New_York";
    readonly America_Chicago: "America/Chicago";
    readonly America_Denver: "America/Denver";
    readonly America_Los_Angeles: "America/Los_Angeles";
    readonly America_Anchorage: "America/Anchorage";
    readonly America_Toronto: "America/Toronto";
    readonly America_Vancouver: "America/Vancouver";
    readonly America_Mexico_City: "America/Mexico_City";
    readonly America_Sao_Paulo: "America/Sao_Paulo";
    readonly America_Buenos_Aires: "America/Buenos_Aires";
    readonly America_Bogota: "America/Bogota";
    readonly America_Santiago: "America/Santiago";
    readonly Europe_London: "Europe/London";
    readonly Europe_Paris: "Europe/Paris";
    readonly Europe_Berlin: "Europe/Berlin";
    readonly Europe_Madrid: "Europe/Madrid";
    readonly Europe_Rome: "Europe/Rome";
    readonly Europe_Amsterdam: "Europe/Amsterdam";
    readonly Europe_Zurich: "Europe/Zurich";
    readonly Europe_Brussels: "Europe/Brussels";
    readonly Europe_Stockholm: "Europe/Stockholm";
    readonly Europe_Oslo: "Europe/Oslo";
    readonly Europe_Helsinki: "Europe/Helsinki";
    readonly Europe_Warsaw: "Europe/Warsaw";
    readonly Europe_Prague: "Europe/Prague";
    readonly Europe_Vienna: "Europe/Vienna";
    readonly Europe_Athens: "Europe/Athens";
    readonly Europe_Bucharest: "Europe/Bucharest";
    readonly Europe_Istanbul: "Europe/Istanbul";
    readonly Europe_Moscow: "Europe/Moscow";
    readonly Europe_Dublin: "Europe/Dublin";
    readonly Europe_Lisbon: "Europe/Lisbon";
    readonly Asia_Dubai: "Asia/Dubai";
    readonly Asia_Riyadh: "Asia/Riyadh";
    readonly Asia_Tehran: "Asia/Tehran";
    readonly Asia_Kolkata: "Asia/Kolkata";
    readonly Asia_Dhaka: "Asia/Dhaka";
    readonly Asia_Bangkok: "Asia/Bangkok";
    readonly Asia_Singapore: "Asia/Singapore";
    readonly Asia_Hong_Kong: "Asia/Hong_Kong";
    readonly Asia_Shanghai: "Asia/Shanghai";
    readonly Asia_Tokyo: "Asia/Tokyo";
    readonly Asia_Seoul: "Asia/Seoul";
    readonly Asia_Taipei: "Asia/Taipei";
    readonly Asia_Jakarta: "Asia/Jakarta";
    readonly Asia_Manila: "Asia/Manila";
    readonly Asia_Karachi: "Asia/Karachi";
    readonly Asia_Jerusalem: "Asia/Jerusalem";
    readonly Asia_Yerevan: "Asia/Yerevan";
    readonly Asia_Tbilisi: "Asia/Tbilisi";
    readonly Asia_Baku: "Asia/Baku";
    readonly Africa_Cairo: "Africa/Cairo";
    readonly Africa_Lagos: "Africa/Lagos";
    readonly Africa_Johannesburg: "Africa/Johannesburg";
    readonly Africa_Nairobi: "Africa/Nairobi";
    readonly Africa_Casablanca: "Africa/Casablanca";
    readonly Australia_Sydney: "Australia/Sydney";
    readonly Australia_Melbourne: "Australia/Melbourne";
    readonly Australia_Perth: "Australia/Perth";
    readonly Australia_Brisbane: "Australia/Brisbane";
    readonly Pacific_Auckland: "Pacific/Auckland";
    readonly Pacific_Honolulu: "Pacific/Honolulu";
    readonly UTC: "UTC";
  };

  export const Weekday: {
    readonly Sunday: "Sunday";
    readonly Monday: "Monday";
    readonly Tuesday: "Tuesday";
    readonly Wednesday: "Wednesday";
    readonly Thursday: "Thursday";
    readonly Friday: "Friday";
    readonly Saturday: "Saturday";
  };

  export interface Classify {
    classification: ClassificationType;
    classificationReasoning: string;
    tags?: string[];
  }

  export interface Enforce {
    enforcementAction: EnforcementActionType;
    enforcementReason: string;
  }

  export function productive(reason: string, tags?: string[]): Classify;
  export function distracting(reason: string, tags?: string[]): Classify;
  export function neutral(reason: string, tags?: string[]): Classify;
  export function block(reason: string): Enforce;
  export function allow(reason: string): Enforce;
  export function pause(reason: string): Enforce;

  /**
   * Summary of time spent in a specific period (e.g., today, this hour).
   */
  export interface TimeSummary {
    /** 
     * Overall productivity score for this period, ranging from 0 to 100.
     * Higher score indicates more time spent on productive activities.
     */
    readonly focusScore: number;
    
    /** Total minutes classified as productive during this period. */
    readonly productiveMinutes: Minutes;
    
    /** Total minutes classified as distracting during this period. */
    readonly distractingMinutes: Minutes;
  }

  /**
   * Insights and duration metrics specific to the currently active application or website.
   */
  export interface CurrentUsage {
    /** Total minutes spent on this specific app/site today. */
    readonly usedToday: Minutes;
    
    /** Number of times this specific app/site was blocked today. */
    readonly blocks: number;
    
    /** 
     * Minutes elapsed since the last block event for this app/site.
     * Returns null if it hasn't been blocked today.
     */
    readonly sinceBlock: Minutes | null;
    
    /** 
     * Minutes actually spent using this app/site since it was last blocked.
     * Returns null if it hasn't been blocked today.
     */
    readonly usedSinceBlock: Minutes | null;
    
    /**
     * Calculates how many minutes were spent on this specific app/site 
     * within the given sliding window of minutes.
     * 
     * @param minutes - The sliding window size in minutes (e.g., 60 for the last hour).
     * @returns Minutes spent on this app/site in that window.
     */
    last(minutes: number): number;
  }

  /**
   * The global runtime context available to your custom rules.
   */
  export interface Runtime {
    /** Aggregate time and score metrics for the entire day. */
    readonly today: TimeSummary;
    
    /** Aggregate time and score metrics for the current hour. */
    readonly hour: TimeSummary;
    
    /** Real-time metadata and metrics for the currently active app or website. */
    readonly usage: Usage;
    
    /** Time utilities bound to specific timezones. */
    readonly time: {
      /** Returns a Date object for the current time in the given timezone. */
      now(timezone?: string): Date;
      /** Returns the current day of the week in the given timezone. */
      day(timezone?: string): WeekdayType;
    };
  }

  /** The global runtime instance. */
  export const runtime: Runtime;

  /**
   * Real-time metadata about the active application or website.
   */
  export interface Usage {
    /** Name of the desktop application (e.g., "Chrome", "Slack"). */
    readonly app: string;
    
    /** Active window title. */
    readonly title: string;
    
    /** Root domain of the website (e.g., "youtube.com"), empty for desktop apps. */
    readonly domain: string;
    
    /** Full hostname of the website (e.g., "www.youtube.com"), empty for desktop apps. */
    readonly host: string;
    
    /** URL path (e.g., "/watch"), empty for desktop apps. */
    readonly path: string;
    
    /** Complete URL, empty for desktop apps. */
    readonly url: string;
    
    /** Current classification of this app/site before custom rules run. */
    readonly classification: string;
    
    /** Granular usage durations and limits for this specific app/site. */
    readonly current: CurrentUsage;
  }
}
`;
