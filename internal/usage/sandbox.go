package usage

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/evanw/esbuild/pkg/api"
	v8 "rogchap.com/v8go"
)

// classificationResult is returned from the classify function.
type classificationResult struct {
	Classification          string   `json:"classification"`
	ClassificationReasoning string   `json:"classificationReasoning"`
	Tags                    []string `json:"tags"`
}

// enforcement is returned from the enforcement function.
type enforcement struct {
	EnforcementAction string `json:"enforcementAction"`
	EnforcementReason string `json:"enforcementReason"`
}

// sandbox executes user-defined JavaScript rules using V8
type sandbox struct {
	isolate *v8.Isolate
	code    string

	logs []string
}

// newSandbox creates a new V8 sandbox with the given JavaScript code
func newSandbox(code string) (*sandbox, error) {
	return &sandbox{
		isolate: v8.NewIsolate(),
		code:    code,
	}, nil
}

func formatEsbuildErrors(errors []api.Message) string {
	if len(errors) == 0 {
		return ""
	}

	messages := api.FormatMessages(errors, api.FormatMessagesOptions{
		Kind:  api.ErrorMessage,
		Color: false,
	})

	return strings.Join(messages, "\n")
}

// prepareScript transpiles TypeScript and adds global function exports, console polyfill, and now() helper
func prepareScript(code string) (string, error) {
	// Transpile user code with CommonJS format to handle export statements
	// Use ES2016 target to transpile async/await to generators which can run synchronously
	result := api.Transform(code, api.TransformOptions{
		Loader: api.LoaderTS,
		Target: api.ES2016,
		Format: api.FormatCommonJS,
	})

	if len(result.Errors) > 0 {
		return "", fmt.Errorf("failed to transpile script: %s", formatEsbuildErrors(result.Errors))
	}

	transpiledCode := string(result.Code)

	// Wrap the transpiled code with CommonJS environment and expose functions to globalThis
	preparedScript := fmt.Sprintf(`
// Define global constants for user scripts
var EnforcementAction = {
	None: "none",
	Block: "block",
	Paused: "paused",
	Allow: "allow"
};

var Classification = {
	Productive: "productive",
	Distracting: "distracting",
	Neutral: "neutral",
	System: "system"
};

var Weekday = {
	Sunday: "Sunday",
	Monday: "Monday",
	Tuesday: "Tuesday",
	Wednesday: "Wednesday",
	Thursday: "Thursday",
	Friday: "Friday",
	Saturday: "Saturday"
};

var Timezone = {
	// Americas
	America_New_York: "America/New_York",
	America_Chicago: "America/Chicago",
	America_Denver: "America/Denver",
	America_Los_Angeles: "America/Los_Angeles",
	America_Anchorage: "America/Anchorage",
	America_Toronto: "America/Toronto",
	America_Vancouver: "America/Vancouver",
	America_Mexico_City: "America/Mexico_City",
	America_Sao_Paulo: "America/Sao_Paulo",
	America_Buenos_Aires: "America/Buenos_Aires",
	America_Bogota: "America/Bogota",
	America_Santiago: "America/Santiago",
	// Europe
	Europe_London: "Europe/London",
	Europe_Paris: "Europe/Paris",
	Europe_Berlin: "Europe/Berlin",
	Europe_Madrid: "Europe/Madrid",
	Europe_Rome: "Europe/Rome",
	Europe_Amsterdam: "Europe/Amsterdam",
	Europe_Zurich: "Europe/Zurich",
	Europe_Brussels: "Europe/Brussels",
	Europe_Stockholm: "Europe/Stockholm",
	Europe_Oslo: "Europe/Oslo",
	Europe_Helsinki: "Europe/Helsinki",
	Europe_Warsaw: "Europe/Warsaw",
	Europe_Prague: "Europe/Prague",
	Europe_Vienna: "Europe/Vienna",
	Europe_Athens: "Europe/Athens",
	Europe_Bucharest: "Europe/Bucharest",
	Europe_Istanbul: "Europe/Istanbul",
	Europe_Moscow: "Europe/Moscow",
	Europe_Dublin: "Europe/Dublin",
	Europe_Lisbon: "Europe/Lisbon",
	// Asia
	Asia_Dubai: "Asia/Dubai",
	Asia_Riyadh: "Asia/Riyadh",
	Asia_Tehran: "Asia/Tehran",
	Asia_Kolkata: "Asia/Kolkata",
	Asia_Dhaka: "Asia/Dhaka",
	Asia_Bangkok: "Asia/Bangkok",
	Asia_Singapore: "Asia/Singapore",
	Asia_Hong_Kong: "Asia/Hong_Kong",
	Asia_Shanghai: "Asia/Shanghai",
	Asia_Tokyo: "Asia/Tokyo",
	Asia_Seoul: "Asia/Seoul",
	Asia_Taipei: "Asia/Taipei",
	Asia_Jakarta: "Asia/Jakarta",
	Asia_Manila: "Asia/Manila",
	Asia_Karachi: "Asia/Karachi",
	Asia_Jerusalem: "Asia/Jerusalem",
	Asia_Yerevan: "Asia/Yerevan",
	Asia_Tbilisi: "Asia/Tbilisi",
	Asia_Baku: "Asia/Baku",
	// Africa
	Africa_Cairo: "Africa/Cairo",
	Africa_Lagos: "Africa/Lagos",
	Africa_Johannesburg: "Africa/Johannesburg",
	Africa_Nairobi: "Africa/Nairobi",
	Africa_Casablanca: "Africa/Casablanca",
	// Oceania
	Australia_Sydney: "Australia/Sydney",
	Australia_Melbourne: "Australia/Melbourne",
	Australia_Perth: "Australia/Perth",
	Australia_Brisbane: "Australia/Brisbane",
	Pacific_Auckland: "Pacific/Auckland",
	Pacific_Honolulu: "Pacific/Honolulu",
	// UTC
	UTC: "UTC"
};

var exports = {};
var module = { exports: exports };

%s

// Expose exported functions to globalThis
// Check both module.exports and exports for functions
var _exported = module.exports || exports;
if (_exported && typeof _exported.classify === 'function') { globalThis.__classify = _exported.classify; }
if (_exported && typeof _exported.enforcement === 'function') { globalThis.__enforcement = _exported.enforcement; }
// Also check for top-level function declarations (non-exported)
if (typeof classify === 'function') { globalThis.__classify = classify; }
if (typeof enforcement === 'function') { globalThis.__enforcement = enforcement; }

// Polyfill console
if (typeof console === 'undefined') {
	globalThis.console = {
		log: __console_log,
		info: __console_log,
		warn: __console_log,
		error: __console_log,
		debug: __console_log
	};
} else {
	console.log = __console_log;
	console.info = __console_log;
	console.warn = __console_log;
	console.error = __console_log;
	console.debug = __console_log;
}

/**
 * Returns a Date object for the current time in the specified IANA timezone.
 * Use Timezone.* constants for autocomplete, or pass any valid IANA timezone string.
 * If no timezone is provided or the string is invalid, uses local time.
 * @param {string} [timezone] - IANA timezone (e.g. Timezone.Europe_London, 'America/New_York')
 * @returns {Date}
 */
function now(timezone) {
    const ts = __getShiftedTimestamp(timezone);
    return new Date(ts);
}

/**
 * Returns the day of the week in the specified IANA timezone.
 * @param {string} [timezone] - IANA timezone (e.g. Timezone.Asia_Tokyo)
 * @returns {string}
 */
function dayOfWeek(timezone) {
    const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
    return days[now(timezone).getDay()];
}

var __currentDay = dayOfWeek();
var IsMonday = __currentDay === "Monday";
var IsTuesday = __currentDay === "Tuesday";
var IsWednesday = __currentDay === "Wednesday";
var IsThursday = __currentDay === "Thursday";
var IsFriday = __currentDay === "Friday";
var IsSaturday = __currentDay === "Saturday";
var IsSunday = __currentDay === "Sunday";
var IsWeekday = !IsSaturday && !IsSunday;
var IsWeekend = IsSaturday || IsSunday;
`, transpiledCode)

	return preparedScript, nil
}

// setupContext prepares the V8 context with globals like __console_log and __getShiftedTimestamp
func (s *sandbox) setupContext(ctx sandboxContext, v8ctx *v8.Context) error {
	global := v8ctx.Global()

	// Inject __getShiftedTimestamp function
	cb := v8.NewFunctionTemplate(s.isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		var loc *time.Location
		var err error

		if len(args) > 0 && args[0].IsString() {
			loc, err = time.LoadLocation(args[0].String())
		}

		// Default to local if not found or error or not provided
		if loc == nil || err != nil {
			loc = time.Local
		}

		var t time.Time
		if ctx.Now != nil {
			t = ctx.Now(loc)
		} else {
			t = time.Now().In(loc)
		}

		// Shift time to appear as Local time but with target wall clock values
		year, month, day := t.Date()
		hour, min, sec := t.Clock()
		nsec := t.Nanosecond()

		shifted := time.Date(year, month, day, hour, min, sec, nsec, time.Local)
		ts := shifted.UnixMilli()

		val, _ := v8.NewValue(s.isolate, float64(ts))
		return val
	})

	fn := cb.GetFunction(v8ctx)
	if err := global.Set("__getShiftedTimestamp", fn); err != nil {
		return fmt.Errorf("failed to set __getShiftedTimestamp function: %w", err)
	}

	// Inject console.log
	consoleCb := v8.NewFunctionTemplate(s.isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		var parts []string
		for _, arg := range args {
			parts = append(parts, arg.String())
		}
		s.logs = append(s.logs, strings.Join(parts, " "))

		return nil
	})

	consoleFn := consoleCb.GetFunction(v8ctx)
	if err := global.Set("__console_log", consoleFn); err != nil {
		return fmt.Errorf("failed to set __console_log function: %w", err)
	}

	// Inject __minutesUsedInPeriod function (appName, hostname, minutes) -> int64
	if ctx.MinutesUsedInPeriod != nil {
		usageCb := v8.NewFunctionTemplate(s.isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
			args := info.Args()
			if len(args) < 3 {
				val, _ := v8.NewValue(s.isolate, int32(0))
				return val
			}

			appName := args[0].String()
			hostname := args[1].String()
			minutes := int64(args[2].Integer())

			result, err := ctx.MinutesUsedInPeriod(appName, hostname, minutes)
			if err != nil {
				slog.Debug("failed to query minutes used", "error", err)
				val, _ := v8.NewValue(s.isolate, int32(0))
				return val
			}

			val, _ := v8.NewValue(s.isolate, int32(result))
			return val
		})

		usageFn := usageCb.GetFunction(v8ctx)
		if err := global.Set("__minutesUsedInPeriod", usageFn); err != nil {
			return fmt.Errorf("failed to set __minutesUsedInPeriod function: %w", err)
		}
	}

	return nil
}

// executeFunction runs the prepared script and then calls the specified function with context
func (s *sandbox) executeFunction(v8ctx *v8.Context, preparedScript string, functionName string, ctx sandboxContext) (string, error) {
	// Run the prepared script to define functions
	_, err := v8ctx.RunScript(preparedScript, "user_rules.js")
	if err != nil {
		return "", fmt.Errorf("failed to execute user script: %w", err)
	}

	// Check if the function exists
	global := v8ctx.Global()
	funcVal, err := global.Get(functionName)
	if err != nil {
		return "", fmt.Errorf("failed to get %s function: %w", functionName, err)
	}

	if funcVal.IsUndefined() || funcVal.IsNull() {
		// Function not defined - return empty
		return "", nil
	}

	// Marshal context to JSON
	ctxJSON, err := json.Marshal(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to marshal context: %w", err)
	}

	// Call the function
	// Expose today/hour globals and add usage.duration.last helper.
	callScript := fmt.Sprintf(`
		(function() {
			const rawCtx = %s;
			Object.defineProperty(globalThis, 'today', {
				value: Object.freeze(rawCtx.today || {}),
				writable: false,
				configurable: true
			});
			Object.defineProperty(globalThis, 'hour', {
				value: Object.freeze(rawCtx.hour || {}),
				writable: false,
				configurable: true
			});

			const ctx = {
				usage: rawCtx.usage || {}
			};

			if (!ctx.usage) {
				ctx.usage = {};
			}

			if (!ctx.usage.meta) {
				ctx.usage.meta = {};
			}

			if (!ctx.usage.duration) {
				ctx.usage.duration = {};
			}

			// Add last method to usage.duration.
			if (typeof __minutesUsedInPeriod === 'function') {
				ctx.usage.duration.last = function(minutes) {
					return __minutesUsedInPeriod(ctx.usage.meta.appName, ctx.usage.meta.host, minutes);
				};
			} else {
				ctx.usage.duration.last = function(minutes) { return 0; };
			}

			const result = %s(ctx);
			if (result === undefined || result === null) {
				return undefined;
			}
			return JSON.stringify(result);
		})()
	`, string(ctxJSON), functionName)

	resultVal, err := v8ctx.RunScript(callScript, "call_function.js")
	if err != nil {
		return "", fmt.Errorf("failed to call %s function: %w", functionName, err)
	}

	if resultVal == nil || resultVal.IsUndefined() || resultVal.IsNull() {
		return "", nil
	}

	resultJSON := resultVal.String()
	if resultJSON == "null" || resultJSON == "undefined" {
		return "", nil
	}

	return resultJSON, nil
}

// close releases V8 resources
func (s *sandbox) close() {
	if s.isolate != nil {
		s.isolate.Dispose()
		s.isolate = nil
	}
}

// invokeClassify executes the classify function and returns the result
// Returns nil if the function returns undefined
func (s *sandbox) invokeClassify(ctx sandboxContext) (*classificationResult, []string, error) {
	// Prepare script with function exports and helpers
	preparedScript, err := prepareScript(s.code)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to prepare script: %w", err)
	}

	v8ctx := v8.NewContext(s.isolate)
	defer s.close()

	// Setup V8 context with __console_log and __getShiftedTimestamp
	if err := s.setupContext(ctx, v8ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to setup context: %w", err)
	}

	// Execute the function
	resultJSON, err := s.executeFunction(v8ctx, preparedScript, "__classify", ctx)
	if err != nil {
		return nil, s.logs, fmt.Errorf("failed to execute classify: %w", err)
	}

	var decision classificationResult

	if resultJSON == "" {
		return nil, s.logs, nil
	}

	if err := json.Unmarshal([]byte(resultJSON), &decision); err != nil {
		return nil, s.logs, fmt.Errorf("failed to parse classification decision: %w", err)
	}

	return &decision, s.logs, nil
}

// invokeEnforcement executes the enforcement function and returns the result.
// Returns nil if the function returns undefined
func (s *sandbox) invokeEnforcement(ctx sandboxContext) (*enforcement, []string, error) {
	// Prepare script with function exports and helpers
	preparedScript, err := prepareScript(s.code)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to prepare script: %w", err)
	}

	v8ctx := v8.NewContext(s.isolate)
	defer s.close()

	// Setup V8 context with __console_log and __getShiftedTimestamp
	if err := s.setupContext(ctx, v8ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to setup context: %w", err)
	}

	// Execute the function
	resultJSON, err := s.executeFunction(v8ctx, preparedScript, "__enforcement", ctx)
	if err != nil {
		return nil, s.logs, fmt.Errorf("failed to execute enforcement: %w", err)
	}

	if resultJSON == "" {
		return nil, s.logs, nil
	}

	var decision enforcement
	if err := json.Unmarshal([]byte(resultJSON), &decision); err != nil {
		return nil, s.logs, fmt.Errorf("failed to parse enforcement decision: %w", err)
	}

	return &decision, s.logs, nil
}
