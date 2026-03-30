package usage

type UsageContributor struct{}

func NewUsageContributor() *UsageContributor {
	return &UsageContributor{}
}

func (c *UsageContributor) Name() string {
	return "usage"
}

func (c *UsageContributor) TypesDefinition() string {
	return `declare module "@focusd/runtime" {
  import { WeekdayType, Timezone } from "@focusd/core";

  export type ClassificationType = "unknown" | "productive" | "distracting" | "neutral" | "system";
  export type EnforcementActionType = "none" | "block" | "paused" | "allow";
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

  export interface TimeSummary {
    readonly focusScore: number;
    readonly productiveMinutes: Minutes;
    readonly distractingMinutes: Minutes;
  }

  export interface CurrentUsage {
    readonly usedToday: Minutes;
    readonly blocks: number;
    readonly sinceBlock: Minutes | null;
    readonly usedSinceBlock: Minutes | null;
    last(minutes: number): number;
  }

  export interface Runtime {
    readonly today: TimeSummary;
    readonly hour: TimeSummary;
    readonly usage: Usage;
    readonly time: {
      now(timezone?: string): Date;
      day(timezone?: string): WeekdayType;
    };
  }

  export const runtime: Runtime;

  export interface Usage {
    readonly app: string;
    readonly title: string;
    readonly domain: string;
    readonly host: string;
    readonly path: string;
    readonly url: string;
    readonly classification: string;
    readonly current: CurrentUsage;
  }
}`
}

func (c *UsageContributor) PolyfillSource() string {
	return `
var Classification = Object.freeze({
	Unknown: "unknown",
	Productive: "productive",
	Distracting: "distracting",
	Neutral: "neutral",
	System: "system"
});

var EnforcementAction = Object.freeze({
	None: "none",
	Block: "block",
	Paused: "paused",
	Allow: "allow"
});

function productive(reason, tags) {
	return { classification: "productive", classificationReasoning: reason, tags: tags };
}
function distracting(reason, tags) {
	return { classification: "distracting", classificationReasoning: reason, tags: tags };
}
function neutral(reason, tags) {
	return { classification: "neutral", classificationReasoning: reason, tags: tags };
}
function block(reason) {
	return { enforcementAction: "block", enforcementReason: reason };
}
function allow(reason) {
	return { enforcementAction: "allow", enforcementReason: reason };
}
function pause(reason) {
	return { enforcementAction: "paused", enforcementReason: reason };
}

if (typeof globalThis.__modules === 'undefined') {
    globalThis.__modules = {};
}

globalThis.__modules["@focusd/runtime"] = {
	Classification: Classification,
	EnforcementAction: EnforcementAction,
	productive: productive,
	distracting: distracting,
	neutral: neutral,
	block: block,
	allow: allow,
	pause: pause,
    // Add pass-throughs from core for backwards compatibility
    Timezone: globalThis.Timezone,
    Weekday: globalThis.Weekday,
	get runtime() {
		return globalThis.__focusd_runtime_context || {
			today: { focusScore: 0, productiveMinutes: 0, distractingMinutes: 0 },
			hour: { focusScore: 0, productiveMinutes: 0, distractingMinutes: 0 },
			time: {
				now: function(tz) { return globalThis.__modules["@focusd/core"].time.now(tz); },
				day: function(tz) { return globalThis.__modules["@focusd/core"].time.day(tz); }
			}
		};
	}
};
Object.freeze(globalThis.__modules["@focusd/runtime"].Classification);
Object.freeze(globalThis.__modules["@focusd/runtime"].EnforcementAction);

// Setup the single custom require polyfill handler if it hasn't been done yet
if (typeof globalThis.require === 'undefined') {
	globalThis.require = function(specifier) {
        if (globalThis.__modules && globalThis.__modules[specifier]) {
            return globalThis.__modules[specifier];
        }
		throw new Error("Unsupported import: " + specifier + ". Only mapped modules are available.");
	};
}

// Wrapper execution functions that unpack the serialized context and assign global state
globalThis.__classify_wrapper = function(ctx) {
	__hydrateContext(ctx);
	if (typeof globalThis.classify !== 'function' && typeof globalThis.__classify !== 'function') {
		return undefined;
	}
	var fn = globalThis.classify || globalThis.__classify;
	return fn();
}

globalThis.__enforcement_wrapper = function(ctx) {
	__hydrateContext(ctx);
	if (typeof globalThis.enforcement !== 'function' && typeof globalThis.__enforcement !== 'function') {
		return undefined;
	}
	var fn = globalThis.enforcement || globalThis.__enforcement;
	return fn();
}

function __hydrateContext(rawCtx) {
	var u = rawCtx.usage || {};
	var meta = u.meta || {};
	var ins = u.insights || {};
	var cur = ins.current || {};
	var dur = cur.duration || {};
	var blk = cur.blocks || {};

	// Create a unique closure to capture meta values for lastFn so they don't leak across calls
	var appName = meta.appName || "";
	var host = meta.host || "";
	var lastFn = (typeof __minutesUsedInPeriod === 'function')
		? function(m) { return __minutesUsedInPeriod(appName, host, m); }
		: function() { return 0; };

	var ctxObj = {
		app: meta.appName || "",
		title: meta.title || "",
		domain: meta.domain || "",
		host: meta.host || "",
		path: meta.path || "",
		url: meta.url || "",
		classification: meta.classification || "",
		current: {
			usedToday: dur.today || 0,
			blocks: blk.count || 0,
			sinceBlock: dur.sinceLastBlock != null ? dur.sinceLastBlock : null,
			usedSinceBlock: dur.usedSinceLastBlock != null ? dur.usedSinceLastBlock : null,
			last: lastFn
		}
	};

	globalThis.__focusd_runtime_context = {
		today: ins.today || { focusScore: 0, productiveMinutes: 0, distractingMinutes: 0 },
		hour: ins.hour || { focusScore: 0, productiveMinutes: 0, distractingMinutes: 0 },
		time: {
			now: function(tz) { return globalThis.__modules["@focusd/core"].time.now(tz); },
			day: function(tz) { return globalThis.__modules["@focusd/core"].time.day(tz); }
		},
		usage: ctxObj
	};
}
`
}
