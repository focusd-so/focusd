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

// countryTimezones maps country codes to their primary timezone
var countryTimezones = map[string]string{
	// A
	"AD": "Europe/Andorra",       // Andorra
	"AE": "Asia/Dubai",           // United Arab Emirates
	"AF": "Asia/Kabul",           // Afghanistan
	"AG": "America/Antigua",      // Antigua and Barbuda
	"AI": "America/Anguilla",     // Anguilla
	"AL": "Europe/Tirane",        // Albania
	"AM": "Asia/Yerevan",         // Armenia
	"AO": "Africa/Luanda",        // Angola
	"AQ": "Antarctica/McMurdo",   // Antarctica
	"AR": "America/Buenos_Aires", // Argentina
	"AS": "Pacific/Pago_Pago",    // American Samoa
	"AT": "Europe/Vienna",        // Austria
	"AU": "Australia/Sydney",     // Australia
	"AW": "America/Aruba",        // Aruba
	"AX": "Europe/Mariehamn",     // Åland Islands
	"AZ": "Asia/Baku",            // Azerbaijan
	// B
	"BA": "Europe/Sarajevo",       // Bosnia and Herzegovina
	"BB": "America/Barbados",      // Barbados
	"BD": "Asia/Dhaka",            // Bangladesh
	"BE": "Europe/Brussels",       // Belgium
	"BF": "Africa/Ouagadougou",    // Burkina Faso
	"BG": "Europe/Sofia",          // Bulgaria
	"BH": "Asia/Bahrain",          // Bahrain
	"BI": "Africa/Bujumbura",      // Burundi
	"BJ": "Africa/Porto-Novo",     // Benin
	"BL": "America/St_Barthelemy", // Saint Barthélemy
	"BM": "Atlantic/Bermuda",      // Bermuda
	"BN": "Asia/Brunei",           // Brunei
	"BO": "America/La_Paz",        // Bolivia
	"BQ": "America/Kralendijk",    // Caribbean Netherlands
	"BR": "America/Sao_Paulo",     // Brazil
	"BS": "America/Nassau",        // Bahamas
	"BT": "Asia/Thimphu",          // Bhutan
	"BV": "Europe/Oslo",           // Bouvet Island (uses Norway timezone)
	"BW": "Africa/Gaborone",       // Botswana
	"BY": "Europe/Minsk",          // Belarus
	"BZ": "America/Belize",        // Belize
	// C
	"CA": "America/Toronto",     // Canada
	"CC": "Indian/Cocos",        // Cocos (Keeling) Islands
	"CD": "Africa/Kinshasa",     // Democratic Republic of the Congo
	"CF": "Africa/Bangui",       // Central African Republic
	"CG": "Africa/Brazzaville",  // Republic of the Congo
	"CH": "Europe/Zurich",       // Switzerland
	"CI": "Africa/Abidjan",      // Côte d'Ivoire
	"CK": "Pacific/Rarotonga",   // Cook Islands
	"CL": "America/Santiago",    // Chile
	"CM": "Africa/Douala",       // Cameroon
	"CN": "Asia/Shanghai",       // China
	"CO": "America/Bogota",      // Colombia
	"CR": "America/Costa_Rica",  // Costa Rica
	"CU": "America/Havana",      // Cuba
	"CV": "Atlantic/Cape_Verde", // Cape Verde
	"CW": "America/Curacao",     // Curaçao
	"CX": "Indian/Christmas",    // Christmas Island
	"CY": "Asia/Nicosia",        // Cyprus
	"CZ": "Europe/Prague",       // Czech Republic
	// D
	"DE": "Europe/Berlin",         // Germany
	"DJ": "Africa/Djibouti",       // Djibouti
	"DK": "Europe/Copenhagen",     // Denmark
	"DM": "America/Dominica",      // Dominica
	"DO": "America/Santo_Domingo", // Dominican Republic
	"DZ": "Africa/Algiers",        // Algeria
	// E
	"EC": "America/Guayaquil",  // Ecuador
	"EE": "Europe/Tallinn",     // Estonia
	"EG": "Africa/Cairo",       // Egypt
	"EH": "Africa/El_Aaiun",    // Western Sahara
	"ER": "Africa/Asmara",      // Eritrea
	"ES": "Europe/Madrid",      // Spain
	"ET": "Africa/Addis_Ababa", // Ethiopia
	// F
	"FI": "Europe/Helsinki",  // Finland
	"FJ": "Pacific/Fiji",     // Fiji
	"FK": "Atlantic/Stanley", // Falkland Islands
	"FM": "Pacific/Chuuk",    // Micronesia
	"FO": "Atlantic/Faroe",   // Faroe Islands
	"FR": "Europe/Paris",     // France
	// G
	"GA": "Africa/Libreville",      // Gabon
	"GB": "Europe/London",          // United Kingdom
	"GD": "America/Grenada",        // Grenada
	"GE": "Asia/Tbilisi",           // Georgia
	"GF": "America/Cayenne",        // French Guiana
	"GG": "Europe/Guernsey",        // Guernsey
	"GH": "Africa/Accra",           // Ghana
	"GI": "Europe/Gibraltar",       // Gibraltar
	"GL": "America/Godthab",        // Greenland
	"GM": "Africa/Banjul",          // Gambia
	"GN": "Africa/Conakry",         // Guinea
	"GP": "America/Guadeloupe",     // Guadeloupe
	"GQ": "Africa/Malabo",          // Equatorial Guinea
	"GR": "Europe/Athens",          // Greece
	"GS": "Atlantic/South_Georgia", // South Georgia and the South Sandwich Islands
	"GT": "America/Guatemala",      // Guatemala
	"GU": "Pacific/Guam",           // Guam
	"GW": "Africa/Bissau",          // Guinea-Bissau
	"GY": "America/Guyana",         // Guyana
	// H
	"HK": "Asia/Hong_Kong",         // Hong Kong
	"HM": "Indian/Kerguelen",       // Heard Island and McDonald Islands
	"HN": "America/Tegucigalpa",    // Honduras
	"HR": "Europe/Zagreb",          // Croatia
	"HT": "America/Port-au-Prince", // Haiti
	"HU": "Europe/Budapest",        // Hungary
	// I
	"ID": "Asia/Jakarta",       // Indonesia
	"IE": "Europe/Dublin",      // Ireland
	"IL": "Asia/Jerusalem",     // Israel
	"IM": "Europe/Isle_of_Man", // Isle of Man
	"IN": "Asia/Kolkata",       // India
	"IO": "Indian/Chagos",      // British Indian Ocean Territory
	"IQ": "Asia/Baghdad",       // Iraq
	"IR": "Asia/Tehran",        // Iran
	"IS": "Atlantic/Reykjavik", // Iceland
	"IT": "Europe/Rome",        // Italy
	// J
	"JE": "Europe/Jersey",   // Jersey
	"JM": "America/Jamaica", // Jamaica
	"JO": "Asia/Amman",      // Jordan
	"JP": "Asia/Tokyo",      // Japan
	// K
	"KE": "Africa/Nairobi",   // Kenya
	"KG": "Asia/Bishkek",     // Kyrgyzstan
	"KH": "Asia/Phnom_Penh",  // Cambodia
	"KI": "Pacific/Tarawa",   // Kiribati
	"KM": "Indian/Comoro",    // Comoros
	"KN": "America/St_Kitts", // Saint Kitts and Nevis
	"KP": "Asia/Pyongyang",   // North Korea
	"KR": "Asia/Seoul",       // South Korea
	"KW": "Asia/Kuwait",      // Kuwait
	"KY": "America/Cayman",   // Cayman Islands
	"KZ": "Asia/Almaty",      // Kazakhstan
	// L
	"LA": "Asia/Vientiane",    // Laos
	"LB": "Asia/Beirut",       // Lebanon
	"LC": "America/St_Lucia",  // Saint Lucia
	"LI": "Europe/Vaduz",      // Liechtenstein
	"LK": "Asia/Colombo",      // Sri Lanka
	"LR": "Africa/Monrovia",   // Liberia
	"LS": "Africa/Maseru",     // Lesotho
	"LT": "Europe/Vilnius",    // Lithuania
	"LU": "Europe/Luxembourg", // Luxembourg
	"LV": "Europe/Riga",       // Latvia
	"LY": "Africa/Tripoli",    // Libya
	// M
	"MA": "Africa/Casablanca",   // Morocco
	"MC": "Europe/Monaco",       // Monaco
	"MD": "Europe/Chisinau",     // Moldova
	"ME": "Europe/Podgorica",    // Montenegro
	"MF": "America/Marigot",     // Saint Martin
	"MG": "Indian/Antananarivo", // Madagascar
	"MH": "Pacific/Majuro",      // Marshall Islands
	"MK": "Europe/Skopje",       // North Macedonia
	"ML": "Africa/Bamako",       // Mali
	"MM": "Asia/Yangon",         // Myanmar
	"MN": "Asia/Ulaanbaatar",    // Mongolia
	"MO": "Asia/Macau",          // Macau
	"MP": "Pacific/Saipan",      // Northern Mariana Islands
	"MQ": "America/Martinique",  // Martinique
	"MR": "Africa/Nouakchott",   // Mauritania
	"MS": "America/Montserrat",  // Montserrat
	"MT": "Europe/Malta",        // Malta
	"MU": "Indian/Mauritius",    // Mauritius
	"MV": "Indian/Maldives",     // Maldives
	"MW": "Africa/Blantyre",     // Malawi
	"MX": "America/Mexico_City", // Mexico
	"MY": "Asia/Kuala_Lumpur",   // Malaysia
	"MZ": "Africa/Maputo",       // Mozambique
	// N
	"NA": "Africa/Windhoek",  // Namibia
	"NC": "Pacific/Noumea",   // New Caledonia
	"NE": "Africa/Niamey",    // Niger
	"NF": "Pacific/Norfolk",  // Norfolk Island
	"NG": "Africa/Lagos",     // Nigeria
	"NI": "America/Managua",  // Nicaragua
	"NL": "Europe/Amsterdam", // Netherlands
	"NO": "Europe/Oslo",      // Norway
	"NP": "Asia/Kathmandu",   // Nepal
	"NR": "Pacific/Nauru",    // Nauru
	"NU": "Pacific/Niue",     // Niue
	"NZ": "Pacific/Auckland", // New Zealand
	// O
	"OM": "Asia/Muscat", // Oman
	// P
	"PA": "America/Panama",       // Panama
	"PE": "America/Lima",         // Peru
	"PF": "Pacific/Tahiti",       // French Polynesia
	"PG": "Pacific/Port_Moresby", // Papua New Guinea
	"PH": "Asia/Manila",          // Philippines
	"PK": "Asia/Karachi",         // Pakistan
	"PL": "Europe/Warsaw",        // Poland
	"PM": "America/Miquelon",     // Saint Pierre and Miquelon
	"PN": "Pacific/Pitcairn",     // Pitcairn Islands
	"PR": "America/Puerto_Rico",  // Puerto Rico
	"PS": "Asia/Gaza",            // Palestine
	"PT": "Europe/Lisbon",        // Portugal
	"PW": "Pacific/Palau",        // Palau
	"PY": "America/Asuncion",     // Paraguay
	// Q
	"QA": "Asia/Qatar", // Qatar
	// R
	"RE": "Indian/Reunion",   // Réunion
	"RO": "Europe/Bucharest", // Romania
	"RS": "Europe/Belgrade",  // Serbia
	"RU": "Europe/Moscow",    // Russia
	"RW": "Africa/Kigali",    // Rwanda
	// S
	"SA": "Asia/Riyadh",           // Saudi Arabia
	"SB": "Pacific/Guadalcanal",   // Solomon Islands
	"SC": "Indian/Mahe",           // Seychelles
	"SD": "Africa/Khartoum",       // Sudan
	"SE": "Europe/Stockholm",      // Sweden
	"SG": "Asia/Singapore",        // Singapore
	"SH": "Atlantic/St_Helena",    // Saint Helena
	"SI": "Europe/Ljubljana",      // Slovenia
	"SJ": "Arctic/Longyearbyen",   // Svalbard and Jan Mayen
	"SK": "Europe/Bratislava",     // Slovakia
	"SL": "Africa/Freetown",       // Sierra Leone
	"SM": "Europe/San_Marino",     // San Marino
	"SN": "Africa/Dakar",          // Senegal
	"SO": "Africa/Mogadishu",      // Somalia
	"SR": "America/Paramaribo",    // Suriname
	"SS": "Africa/Juba",           // South Sudan
	"ST": "Africa/Sao_Tome",       // São Tomé and Príncipe
	"SV": "America/El_Salvador",   // El Salvador
	"SX": "America/Lower_Princes", // Sint Maarten
	"SY": "Asia/Damascus",         // Syria
	"SZ": "Africa/Mbabane",        // Eswatini
	// T
	"TC": "America/Grand_Turk",    // Turks and Caicos Islands
	"TD": "Africa/Ndjamena",       // Chad
	"TF": "Indian/Kerguelen",      // French Southern and Antarctic Lands
	"TG": "Africa/Lome",           // Togo
	"TH": "Asia/Bangkok",          // Thailand
	"TJ": "Asia/Dushanbe",         // Tajikistan
	"TK": "Pacific/Fakaofo",       // Tokelau
	"TL": "Asia/Dili",             // Timor-Leste
	"TM": "Asia/Ashgabat",         // Turkmenistan
	"TN": "Africa/Tunis",          // Tunisia
	"TO": "Pacific/Tongatapu",     // Tonga
	"TR": "Europe/Istanbul",       // Turkey
	"TT": "America/Port_of_Spain", // Trinidad and Tobago
	"TV": "Pacific/Funafuti",      // Tuvalu
	"TW": "Asia/Taipei",           // Taiwan
	"TZ": "Africa/Dar_es_Salaam",  // Tanzania
	// U
	"UA": "Europe/Kiev",        // Ukraine
	"UG": "Africa/Kampala",     // Uganda
	"UM": "Pacific/Wake",       // United States Minor Outlying Islands
	"US": "America/New_York",   // United States
	"UY": "America/Montevideo", // Uruguay
	"UZ": "Asia/Tashkent",      // Uzbekistan
	// V
	"VA": "Europe/Vatican",     // Vatican City
	"VC": "America/St_Vincent", // Saint Vincent and the Grenadines
	"VE": "America/Caracas",    // Venezuela
	"VG": "America/Tortola",    // British Virgin Islands
	"VI": "America/St_Thomas",  // U.S. Virgin Islands
	"VN": "Asia/Ho_Chi_Minh",   // Vietnam
	"VU": "Pacific/Efate",      // Vanuatu
	// W
	"WF": "Pacific/Wallis", // Wallis and Futuna
	"WS": "Pacific/Apia",   // Samoa
	// Y
	"YE": "Asia/Aden",      // Yemen
	"YT": "Indian/Mayotte", // Mayotte
	// Z
	"ZA": "Africa/Johannesburg", // South Africa
	"ZM": "Africa/Lusaka",       // Zambia
	"ZW": "Africa/Harare",       // Zimbabwe
}

// classificationDecision is returned from the classify function
type classificationDecision struct {
	Classification          string   `json:"classification"`
	ClassificationReasoning string   `json:"classificationReasoning"`
	Tags                    []string `json:"tags"`
}

// terminationDecision is returned from the termination function
type terminationDecision struct {
	TerminationMode      string `json:"terminationMode"`
	TerminationReasoning string `json:"terminationReasoning"`
}

// sandboxContext provides context for the current rule execution including usage data and helper functions
type sandboxContext struct {
	// Input data
	AppName        string `json:"appName"`
	ExecutablePath string `json:"executablePath"`

	Hostname       string `json:"hostname"`
	Path           string `json:"path"`
	Domain         string `json:"domain"`
	URL            string `json:"url"`
	Classification string `json:"classification"`

	// Helper pre-computed values
	MinutesSinceLastBlock     *int `json:"minutesSinceLastBlock"`
	MinutesUsedSinceLastBlock *int `json:"minutesUsedSinceLastBlock"`

	// Helper functions
	Now                 func(loc *time.Location) time.Time                                    `json:"-"`
	MinutesUsedInPeriod func(bundleID, hostname string, durationMinutes int64) (int64, error) `json:"-"`
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
var TerminationMode = {
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

var exports = {};
var module = { exports: exports };

%s

// Expose exported functions to globalThis
// Check both module.exports and exports for functions
var _exported = module.exports || exports;
if (_exported && typeof _exported.classify === 'function') { globalThis.__classify = _exported.classify; }
if (_exported && typeof _exported.terminationMode === 'function') { globalThis.__terminationMode = _exported.terminationMode; }
// Also check for top-level function declarations (non-exported)
if (typeof classify === 'function') { globalThis.__classify = classify; }
if (typeof terminationMode === 'function') { globalThis.__terminationMode = terminationMode; }

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
 * Returns a Date object shifted to the specified country's timezone.
 * If no country code is provided or found, uses local time.
 * @param {string} [countryCode] - 2-letter country code (e.g. 'US', 'JP')
 * @returns {Date}
 */
function now(countryCode) {
    const ts = __getShiftedTimestamp(countryCode);
    return new Date(ts);
}

/**
 * Returns the day of the week for the specified country's timezone.
 * @param {string} [countryCode] - 2-letter country code (e.g. 'US', 'JP')
 * @returns {string}
 */
function dayOfWeek(countryCode) {
    const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
    return days[now(countryCode).getDay()];
}
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
			cc := strings.ToUpper(args[0].String())
			if tz, ok := countryTimezones[cc]; ok {
				loc, err = time.LoadLocation(tz)
			}
		}

		// Default to local if not found or error or not provided
		if loc == nil || err != nil {
			loc = time.Local
		}

		now := ctx.Now(loc)

		// Shift time to appear as Local time but with target wall clock values
		year, month, day := now.Date()
		hour, min, sec := now.Clock()
		nsec := now.Nanosecond()

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

	// Inject __minutesUsedInPeriod function (bundleID, hostname, minutes) -> int64
	if ctx.MinutesUsedInPeriod != nil {
		usageCb := v8.NewFunctionTemplate(s.isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
			args := info.Args()
			if len(args) < 3 {
				val, _ := v8.NewValue(s.isolate, int32(0))
				return val
			}

			bundleID := args[0].String()
			hostname := args[1].String()
			minutes := int64(args[2].Integer())

			result, err := ctx.MinutesUsedInPeriod(bundleID, hostname, minutes)
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

	// Inject __minutesSinceLastBlockValue as a pre-computed value
	if ctx.MinutesSinceLastBlock != nil {
		val, _ := v8.NewValue(s.isolate, int32(*ctx.MinutesSinceLastBlock))
		if err := global.Set("__minutesSinceLastBlockValue", val); err != nil {
			return fmt.Errorf("failed to set __minutesSinceLastBlockValue: %w", err)
		}
	}

	// Inject __minutesUsedSinceLastBlockValue as a pre-computed value
	if ctx.MinutesUsedSinceLastBlock != nil {
		val, _ := v8.NewValue(s.isolate, int32(*ctx.MinutesUsedSinceLastBlock))
		if err := global.Set("__minutesUsedSinceLastBlockValue", val); err != nil {
			return fmt.Errorf("failed to set __minutesUsedSinceLastBlockValue: %w", err)
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
	// Add minutesUsedInPeriod as a method on the context object
	callScript := fmt.Sprintf(`
		(function() {
			const ctx = %s;
				// Add minutesUsedInPeriod method to context
			if (typeof __minutesUsedInPeriod === 'function') {
				ctx.minutesUsedInPeriod = function(minutes) {
					return __minutesUsedInPeriod(ctx.bundleID, ctx.hostname, minutes);
				};
			} else {
				ctx.minutesUsedInPeriod = function(minutes) { return 0; };
			}

			// Add minutesSinceLastBlock as a method that returns the pre-computed value
			if (typeof __minutesSinceLastBlockValue === 'number') {
				ctx.minutesSinceLastBlock = __minutesSinceLastBlockValue;
			} else {
				ctx.minutesSinceLastBlock = -1;
			}

			// Add minutesUsedSinceLastBlock as a method that returns the pre-computed value
			if (typeof __minutesUsedSinceLastBlockValue === 'number') {
				ctx.minutesUsedSinceLastBlock = __minutesUsedSinceLastBlockValue;
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
func (s *sandbox) invokeClassify(ctx sandboxContext) (*classificationDecision, []string, error) {
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

	var decision classificationDecision

	if resultJSON == "" {
		return nil, s.logs, nil
	}

	if err := json.Unmarshal([]byte(resultJSON), &decision); err != nil {
		return nil, s.logs, fmt.Errorf("failed to parse classification decision: %w", err)
	}

	return &decision, s.logs, nil
}

// invokeTerminationMode executes the termination function and returns the result
// Returns nil if the function returns undefined
func (s *sandbox) invokeTerminationMode(ctx sandboxContext) (*terminationDecision, error) {
	// Prepare script with function exports and helpers
	preparedScript, err := prepareScript(s.code)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare script: %w", err)
	}

	v8ctx := v8.NewContext(s.isolate)
	defer s.close()

	// Setup V8 context with __console_log and __getShiftedTimestamp
	if err := s.setupContext(ctx, v8ctx); err != nil {
		return nil, fmt.Errorf("failed to setup context: %w", err)
	}

	// Execute the function
	resultJSON, err := s.executeFunction(v8ctx, preparedScript, "__terminationMode", ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute terminationMode: %w", err)
	}

	if resultJSON == "" {
		return nil, nil
	}

	var decision terminationDecision
	if err := json.Unmarshal([]byte(resultJSON), &decision); err != nil {
		return nil, fmt.Errorf("failed to parse termination decision: %w", err)
	}

	return &decision, nil
}
