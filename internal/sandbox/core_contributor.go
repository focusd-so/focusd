package sandbox

import (
	"fmt"
	"time"

	v8 "rogchap.com/v8go"
)

// coreContributor provides fundamental utilities available to all sandboxes
type coreContributor struct{}

func (c *coreContributor) Name() string {
	return "core"
}

func (c *coreContributor) TypesDefinition() string {
	return `declare module "@focusd/core" {
  export type WeekdayType = "Sunday" | "Monday" | "Tuesday" | "Wednesday" | "Thursday" | "Friday" | "Saturday";
  
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

  export interface Time {
    now(timezone?: string): Date;
    day(timezone?: string): WeekdayType;
  }

  export const time: Time;
}
`
}

func (c *coreContributor) PolyfillSource() string {
	return `
var Weekday = Object.freeze({
	Sunday: "Sunday",
	Monday: "Monday",
	Tuesday: "Tuesday",
	Wednesday: "Wednesday",
	Thursday: "Thursday",
	Friday: "Friday",
	Saturday: "Saturday"
});

var Timezone = Object.freeze({
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
	Africa_Cairo: "Africa/Cairo",
	Africa_Lagos: "Africa/Lagos",
	Africa_Johannesburg: "Africa/Johannesburg",
	Africa_Nairobi: "Africa/Nairobi",
	Africa_Casablanca: "Africa/Casablanca",
	Australia_Sydney: "Australia/Sydney",
	Australia_Melbourne: "Australia/Melbourne",
	Australia_Perth: "Australia/Perth",
	Australia_Brisbane: "Australia/Brisbane",
	Pacific_Auckland: "Pacific/Auckland",
	Pacific_Honolulu: "Pacific/Honolulu",
	UTC: "UTC"
});

function __runtimeNow(timezone) {
	const ts = __getShiftedTimestamp(timezone);
	return new Date(ts);
}

function __runtimeDayOfWeek(timezone) {
	const days = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];
	return days[__runtimeNow(timezone).getDay()];
}

// Make core modules available for importing if a module system exists
if (typeof __modules === 'undefined') {
    globalThis.__modules = {};
}

__modules["@focusd/core"] = {
    Timezone: Timezone,
    Weekday: Weekday,
    time: {
        now: __runtimeNow,
        day: __runtimeDayOfWeek
    }
};

// Also inject into global scope for convenience in basic scripts
globalThis.Timezone = Timezone;
globalThis.Weekday = Weekday;
`
}

func (c *coreContributor) RegisterGlobals(iso *v8.Isolate, global *v8.ObjectTemplate) error {
	// Inject __getShiftedTimestamp function
	timeCb := v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
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

		t := time.Now().In(loc)

		// Shift time to appear as Local time but with target wall clock values
		year, month, day := t.Date()
		hour, min, sec := t.Clock()
		nsec := t.Nanosecond()

		shifted := time.Date(year, month, day, hour, min, sec, nsec, time.Local)
		ts := shifted.UnixMilli()

		val, _ := v8.NewValue(iso, float64(ts))
		return val
	})

	if err := global.Set("__getShiftedTimestamp", timeCb); err != nil {
		return fmt.Errorf("failed to set __getShiftedTimestamp function: %w", err)
	}

	return nil
}
