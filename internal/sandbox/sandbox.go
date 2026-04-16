package sandbox

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	v8 "rogchap.com/v8go"
)

// Result contains the output of the sandbox execution and the console logs
type Result struct {
	Output string
	Logs   []string
}

// Sandbox represents a single execution context
type Sandbox struct {
	isolate *v8.Isolate
	global  *v8.ObjectTemplate
	logs    []string
}

// New creates a new V8 sandbox
func New() (*Sandbox, error) {
	isolate := v8.NewIsolate()
	global := v8.NewObjectTemplate(isolate)

	s := &Sandbox{
		isolate: isolate,
		global:  global,
		logs:    make([]string, 0),
	}

	// Inject console.log
	consoleCb := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		var parts []string
		for _, arg := range args {
			parts = append(parts, arg.String())
		}
		s.logs = append(s.logs, strings.Join(parts, " "))
		return nil
	})

	if err := global.Set("__console_log", consoleCb); err != nil {
		s.Close()
		return nil, fmt.Errorf("failed to set __console_log function: %w", err)
	}

	// Internal core contributor registration
	core := &coreContributor{}
	if err := core.RegisterGlobals(isolate, global); err != nil {
		s.Close()
		return nil, fmt.Errorf("failed to register core globals: %w", err)
	}

	return s, nil
}

// RegisterGlobal injects a dynamic Go function into this specific Sandbox execution context
func (s *Sandbox) RegisterGlobal(name string, cb v8.FunctionCallback) error {
	fn := v8.NewFunctionTemplate(s.isolate, cb)
	return s.global.Set(name, fn)
}

// Close releases V8 resources
func (s *Sandbox) Close() {
	if s.isolate != nil {
		s.isolate.Dispose()
		s.isolate = nil
	}
}

// Execute runs the JavaScript code and calls the specified function with JSON-serialized args
func (s *Sandbox) Execute(code string, fnName string, args ...any) (*Result, error) {
	// Transpile user code with CommonJS format to handle export statements
	// Use ES2016 target to transpile async/await to generators which can run synchronously
	result := api.Transform(code, api.TransformOptions{
		Loader: api.LoaderTS,
		Target: api.ES2016,
		Format: api.FormatCommonJS,
	})

	if len(result.Errors) > 0 {
		messages := api.FormatMessages(result.Errors, api.FormatMessagesOptions{
			Kind:  api.ErrorMessage,
			Color: false,
		})
		return nil, fmt.Errorf("failed to transpile script: %s", strings.Join(messages, "\n"))
	}

	transpiledCode := string(result.Code)

	var sb strings.Builder

	// Setup basic exports mechanism for CJS transpiled code
	sb.WriteString(`
var exports = {};
var module = { exports: exports };
globalThis.__modules = globalThis.__modules || {};
`)

	// Inject all contributor polyfills BEFORE the user code
	for _, c := range contributors {
		sb.WriteString(fmt.Sprintf("\n// --- Polyfill %s ---\n", c.Name()))
		sb.WriteString(c.PolyfillSource())
	}

	// Simple polyfill for require() if not already defined by a contributor
	sb.WriteString(`
if (typeof globalThis.require === 'undefined') {
	globalThis.require = function(specifier) {
		if (globalThis.__modules && globalThis.__modules[specifier]) {
			return globalThis.__modules[specifier];
		}
		throw new Error("Unsupported import: " + specifier + ". Only mapped modules are available.");
	};
}
`)

	sb.WriteString("\n// --- User Code ---\n")
	sb.WriteString(transpiledCode)

	sb.WriteString("\n// Expose exported functions to globalThis\n")
	sb.WriteString("var _exported = module.exports || exports;\n")
	sb.WriteString("for (var key in _exported) { if (typeof _exported[key] === 'function') { globalThis[key] = _exported[key]; } }\n")

	preparedScript := sb.String()

	v8ctx := v8.NewContext(s.isolate, s.global)
	defer v8ctx.Close()

	// Polyfill basic console in JS
	_, err := v8ctx.RunScript(`
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
	`, "console_polyfill.js")

	if err != nil {
		return nil, fmt.Errorf("failed to polyfill console: %w", err)
	}

	// Run the prepared script to define functions
	_, err = v8ctx.RunScript(preparedScript, "user_rules.js")
	if err != nil {
		return &Result{Logs: s.logs}, fmt.Errorf("failed to execute user script: %w", err)
	}

	global := v8ctx.Global()
	funcVal, err := global.Get(fnName)
	if err != nil {
		return &Result{Logs: s.logs}, fmt.Errorf("failed to get %s function: %w", fnName, err)
	}

	if funcVal.IsUndefined() || funcVal.IsNull() {
		// Function not defined - return empty
		return &Result{Logs: s.logs}, nil
	}

	// Serialize arguments to JSON
	var jsonArgs []string
	for _, arg := range args {
		b, err := json.Marshal(arg)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal arg to JSON: %w", err)
		}
		jsonArgs = append(jsonArgs, string(b))
	}

	callScript := fmt.Sprintf(`
		(function() {
			var args = [%s];
			var fn = globalThis['%s'];
			if (typeof fn !== 'function') return undefined;
			
			var result = fn.apply(null, args);
			if (result === undefined || result === null) {
				return undefined;
			}
			return JSON.stringify(result);
		})()
	`, strings.Join(jsonArgs, ", "), fnName)

	resultVal, err := v8ctx.RunScript(callScript, "call_function.js")
	if err != nil {
		return &Result{Logs: s.logs}, fmt.Errorf("failed to call %s function: %w", fnName, err)
	}

	if resultVal == nil || resultVal.IsUndefined() || resultVal.IsNull() {
		return &Result{Logs: s.logs}, nil
	}

	resultJSON := resultVal.String()
	if resultJSON == "null" || resultJSON == "undefined" {
		return &Result{Logs: s.logs}, nil
	}

	return &Result{
		Output: resultJSON,
		Logs:   s.logs,
	}, nil
}
