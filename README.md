<div align="center">
  <img src="build/trayicon.png" width="128" alt="Focusd Icon"/>
  <h1>Focusd</h1>
  <p><strong>Stay in flow, ship without distractions.</strong></p>
</div>

Focusd is a privacy-first, macOS-focused distraction blocker aiming to help developers and creators maintain deep work sessions. Rather than manually creating blocklists, Focusd uses LLMs to conditionally restrict distracting apps and URLs, gently nudging you back to your work.

> **Note**: This project is source-available under the **GNU Affero General Public License Version 3 (AGPLv3)**. While you are free to examine, deploy, and learn from its code, you are not permitted to use this source code for closed-source commercial projects without proper licensing.

## Features

- **Smart Blocking**: Uses intelligent categorization to decide if an application or website is distracting.
- **Context Monitoring**: Tracks your active application titles and browser URLs to track your context and activity.
- **Mac Native Experience**: Built with a sleek, translucent, frameless macOS system tray interface and webview.
- **Local Database**: Runs entirely locally using an embedded SQLite/LibSQL database, keeping your data on your device.

## Architecture

Focusd relies on the following technologies:

- **[Wails v3](https://v3.wails.io/)**: The core framework blending a Go backend with a modern web frontend.
- **Go**: Powers native macOS interactions, database interactions, LLM integrations, and core application state.
- **React / TypeScript**: The frontend UI running seamlessly inside a macOS webview.
- **SQLite (LibSQL) & GORM**: Provides robust, local, on-device storage.

## Getting Started (Development)

To build and run Focusd locally, you will need **Go 1.22+** and **Node.js 18+** installed on your macOS machine. You will also need the Wails v3 CLI.

1. **Install the Wails v3 CLI**

   ```bash
   go install github.com/wailsapp/wails/v3/cmd/wails3@latest
   ```

2. **Clone the repository**

   ```bash
   git clone https://github.com/focusd-so/focusd.git
   cd focusd
   ```

3. **Install frontend dependencies**

   ```bash
   cd frontend
   npm install
   cd ..
   ```

4. **Run in development mode**

   ```bash
   wails3 dev
   ```

   This immediately starts the application with hot-reloading enabled for both the Go backend and the frontend React components.

5. **Run the API server separately (optional)**
   If you need to test the backend API proxy separately, you can run the background server command:
   ```bash
   go run ./cmd/main.go serve
   ```
   _Note: Ensure you have your environment variables set appropriately (e.g., `TURSO_DATABASE_URL`, `TURSO_AUTH_TOKEN`, `GEMINI_API_KEY`) before starting the proxy._

## Building and Packaging

To produce a self-contained production binary or macOS `.app` bundle, simply run:

```bash
wails3 build
```

This will compile and create a macOS application inside the `build/bin/` equivalent directory.

Also included is a robust build pipeline located in `.github/workflows` to release `.dmg` files automatically utilizing Taskfile and Github Actions.

## Contributing

We welcome community contributions, from design changes to core functionality upgrades! Please read our [LICENSE](LICENSE) terms carefully. Since we use AGPLv3, any modified versions of Focusd running on a server or distributed over a network must also be made publicly available under the AGPLv3.

If you find a bug or have a suggestion, feel free to open an Issue or a Pull Request.

## License

Focusd is provided under the **GNU Affero General Public License v3**. See the [LICENSE](LICENSE) file for complete details.
