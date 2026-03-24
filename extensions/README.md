# Focusd Browser Extensions

This package uses WXT to build one extension source for Chromium and Firefox.

## Development

```bash
cd extensions
npm install
npm run dev
```

For Firefox:

```bash
npm run dev:firefox
```

## Build

```bash
npm run build
npm run build:firefox
```

The background worker opens a native messaging port to `app.focusd.so`, requests
connection info, and then connects to Focusd websocket endpoint at
`ws://127.0.0.1:50533/extension/ws` with the returned API key.
