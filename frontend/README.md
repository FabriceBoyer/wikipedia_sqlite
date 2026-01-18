# Wikipedia SQLite Frontend

Modern React application built with Vite and TypeScript.

## Development

```bash
# Install dependencies
npm install

# Start dev server (with proxy to Go backend)
npm run dev

# Build for production
npm run build
```

The dev server runs on `http://localhost:5173` and proxies API requests to `http://localhost:9096`.

## Production Build

The build output goes to `../static/` which is served by the Go backend.

```bash
npm run build
```

## Project Structure

```
frontend/
├── src/
│   ├── components/     # React components
│   ├── types/          # TypeScript type definitions
│   ├── utils/          # Utility functions (API client)
│   ├── App.tsx         # Main app component
│   ├── App.css         # App styles
│   ├── main.tsx        # Entry point
│   └── index.css       # Global styles
├── public/             # Static assets
├── index.html          # HTML template
├── vite.config.ts       # Vite configuration
└── package.json        # Dependencies
```

## Features

- **React 19** with TypeScript
- **React Router** for navigation and browser history
- **Vite** for fast development and optimized builds
- **Modern CSS** with component-based styling
- **Type-safe API client** with TypeScript


