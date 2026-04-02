package usage

import (
	"fmt"
	"log/slog"

	"github.com/focusd-so/focusd/internal/sandbox"
	v8 "rogchap.com/v8go"
)

type usageContributor struct {
	svc *Service
}

// NewUsageContributor wraps the usage service into a sandbox contributor
func NewUsageContributor(svc *Service) sandbox.Contributor {
	return &usageContributor{svc: svc}
}

// Name implements sandbox.Contributor
func (c *usageContributor) Name() string {
	return "usage"
}

// TypesDefinition implements sandbox.Contributor
func (c *usageContributor) TypesDefinition() string {
	return `declare module "@focusd/runtime" {
  import { WeekdayType, Timezone, Weekday } from "@focusd/core";
  export { WeekdayType, Timezone, Weekday };

  /**
   * High-level category for the current activity.
   */
  export type ClassificationType = "unknown" | "productive" | "distracting" | "neutral" | "system";

  /**
   * Action the runtime should take for the current activity.
   */
  export type EnforcementActionType = "none" | "block" | "paused" | "allow";

  /**
   * Duration measured in whole minutes.
   */
  export type Minutes = number;

  /**
   * Named constants for available classification values.
   */
  export const Classification: {
    /** Activity could not be confidently categorized. */
    readonly Unknown: "unknown";
    /** Activity is considered helpful for focus or work. */
    readonly Productive: "productive";
    /** Activity is considered distracting. */
    readonly Distracting: "distracting";
    /** Activity is neither clearly productive nor distracting. */
    readonly Neutral: "neutral";
    /** Activity is system-generated and should not affect scoring. */
    readonly System: "system";
  };

  /**
   * Named constants for available enforcement actions.
   */
  export const EnforcementAction: {
    /** No enforcement action is applied. */
    readonly None: "none";
    /** Access is blocked. */
    readonly Block: "block";
    /** Access is temporarily paused. */
    readonly Paused: "paused";
    /** Access is explicitly allowed. */
    readonly Allow: "allow";
  };

  /**
   * Result returned by classification helpers.
   */
  export interface Classify {
    /** Classification category for the active usage. */
    classification: ClassificationType;
    /** Human-readable reason describing why this classification was chosen. */
    classificationReasoning: string;
    /** Optional labels to support filtering, analysis, or reporting. */
    tags?: string[];
  }

  /**
   * Result returned by enforcement helpers.
   */
  export interface Enforce {
    /** Enforcement action that should be taken. */
    enforcementAction: EnforcementActionType;
    /** Human-readable reason describing why this action was chosen. */
    enforcementReason: string;
  }

  /**
   * Marks current activity as productive.
   * @param reason Why the activity is productive.
   * @param tags Optional tags to attach to this decision.
   */
  export function productive(reason: string, tags?: string[]): Classify;

  /**
   * Marks current activity as distracting.
   * @param reason Why the activity is distracting.
   * @param tags Optional tags to attach to this decision.
   */
  export function distracting(reason: string, tags?: string[]): Classify;

  /**
   * Marks current activity as neutral.
   * @param reason Why the activity is neutral.
   * @param tags Optional tags to attach to this decision.
   */
  export function neutral(reason: string, tags?: string[]): Classify;

  /**
   * Blocks access to the current activity.
   * @param reason Why access should be blocked.
   */
  export function block(reason: string): Enforce;

  /**
   * Explicitly allows access to the current activity.
   * @param reason Why access should be allowed.
   */
  export function allow(reason: string): Enforce;

  /**
   * Temporarily pauses access to the current activity.
   * @param reason Why access should be paused.
   */
  export function pause(reason: string): Enforce;

  /**
   * Aggregated focus metrics for a period.
   */
  export interface TimeSummary {
    /** Focus score for the period on a 0-100 scale. */
    readonly focusScore: number;
    /** Minutes classified as productive in the period. */
    readonly productiveMinutes: Minutes;
    /** Minutes classified as distracting in the period. */
    readonly distractingMinutes: Minutes;
  }

  /**
   * Activity totals for the currently active app/site.
   */
  export interface CurrentUsage {
    /** Minutes spent on this app/site today. */
    readonly usedToday: Minutes;
    /** Number of block events for this app/site today. */
    readonly blocks: number;
    /** Minutes elapsed since the most recent block event, if any. */
    readonly sinceBlock: Minutes | null;
    /** Minutes used since the most recent block event, if any. */
    readonly usedSinceBlock: Minutes | null;

    /**
     * Returns minutes of usage during the last N minutes for this app/site.
     * @param minutes Size of lookback window in minutes.
     */
    last(minutes: number): number;
  }

  /**
   * Runtime context available to classify/enforcement scripts.
   */
  export interface Runtime {
    /** Focus metrics aggregated for today. */
    readonly today: TimeSummary;
    /** Focus metrics aggregated for the last hour. */
    readonly hour: TimeSummary;
    /** Metadata for the currently active app/page. */
    readonly usage: Usage;

    /**
     * Time helpers used by scripts.
     */
    readonly time: {
      /**
       * Current date/time in local time or in the provided IANA timezone.
       * @param timezone Optional IANA timezone like "America/New_York".
       */
      now(timezone?: string): Date;

      /**
       * Current weekday in local time or in the provided IANA timezone.
       * @param timezone Optional IANA timezone like "America/New_York".
       */
      day(timezone?: string): WeekdayType;
    };
  }

  /**
   * Runtime context object injected by focusd during script execution.
   */
  export const runtime: Runtime;

  /**
   * Metadata about the currently active app/window/page.
   */
  export interface Usage {
    /** Name of the active application (for example, "Google Chrome"). */
    readonly app: string;
    /** Window or tab title reported by the active application. */
    readonly title: string;
    /** Domain name for browser activity, when available. */
    readonly domain: string;
    /** Host for browser activity, when available. */
    readonly host: string;
    /** URL path for browser activity, when available. */
    readonly path: string;
    /** Full URL for browser activity, when available. */
    readonly url: string;
    /** Existing classification assigned to this activity before script output. */
    readonly classification: string;
    /** Usage metrics scoped to this exact app/site. */
    readonly current: CurrentUsage;
  }
}`
}

// PolyfillSource implements sandbox.Contributor
func (c *usageContributor) PolyfillSource() string {
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

// RegisterGlobals implements sandbox.Contributor and statically provides Usage DB methods
func (c *usageContributor) RegisterGlobals(iso *v8.Isolate, global *v8.ObjectTemplate) error {
	usageCb := v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		if len(args) < 3 {
			val, _ := v8.NewValue(iso, int32(0))
			return val
		}

		appName := args[0].String()
		hostname := args[1].String()
		minutes := int64(args[2].Integer())

		result, err := c.svc.minutesUsedInPeriod(appName, hostname, minutes)
		if err != nil {
			slog.Debug("failed to query minutes used", "error", err)
			val, _ := v8.NewValue(iso, int32(0))
			return val
		}

		val, _ := v8.NewValue(iso, int32(result))
		return val
	})

	if err := global.Set("__minutesUsedInPeriod", usageCb); err != nil {
		return fmt.Errorf("failed to set __minutesUsedInPeriod function: %w", err)
	}

	return nil
}
