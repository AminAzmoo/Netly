# Netly Admin Frontend

A modern React + TypeScript + Tailwind CSS admin interface for Netly network management.

## Features

- **Pure black dot-grid background** with subtle radial pattern
- **Two neon accent colors**: #00E5FF (primary) and #C400FF (secondary)
- **Silver gradient typography** for titles and subtitles
- **VerticalProcessTimeline** component for real-time operation tracking
- **Six main pages**: Dashboard, Devices, Tunnels, Services, Timeline, Settings

## Tech Stack

- React 18
- TypeScript
- Tailwind CSS
- React Router
- Vite

## Getting Started

Install dependencies:
```bash
npm install
```

Run development server:
```bash
npm run dev
```

Build for production:
```bash
npm run build
```

## Project Structure

```
src/
├── components/
│   ├── layout/          # DotGridBackground, HeaderDock, PageShell
│   ├── common/          # CardShell, StatusBadge, VerticalProcessTimeline
│   └── entities/        # DeviceCard, TunnelCard, ServiceCard
├── pages/               # All page components
├── types/               # TypeScript interfaces
└── data/                # Mock data for demo
```

## Key Components

### VerticalProcessTimeline
Displays step-by-step progress for long-running operations with animated states:
- `pending`: dim outline
- `running`: pulsing neon glow
- `done`: filled with checkmark
- `error`: red-tinted with cross icon

### Color Palette
- Background: `#020309`
- Neon A (primary): `#00E5FF`
- Neon B (secondary): `#C400FF`
- Neutrals: grays and silver only

## Design Principles

- No sidebar navigation (header tabs only)
- Glass-morphism cards with subtle borders
- Minimal use of color (only neon accents)
- Real-time progress visualization on entity cards
- Responsive and accessible
