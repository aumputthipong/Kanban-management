import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  compress: true,
  images: {
    formats: ["image/avif", "image/webp"],
  },
  // lucide-react ships every icon as a barrel; rewrite to a per-icon path so
  // unused icons don't get bundled. MUI v7 already ESM tree-shakes; trying to
  // modularize @mui/material breaks subpaths like createTheme/ThemeProvider.
  modularizeImports: {
    "lucide-react": {
      transform: "lucide-react/dist/esm/icons/{{kebabCase member}}",
    },
  },
  // Proxy same-origin "/api/*" calls from the browser to the real backend so
  // the auth cookie is first-party (see lib/constants.ts). NEXT_PUBLIC_API_URL
  // already includes the "/api" suffix, e.g. https://turtask-api.onrender.com/api.
  async rewrites() {
    const backend = process.env.NEXT_PUBLIC_API_URL;
    if (!backend) return [];
    return [{ source: "/api/:path*", destination: `${backend}/:path*` }];
  },
  async redirects() {
    return [
      // /my-tasks was renamed to /my-work in S.1. Permanent redirect so
      // existing bookmarks and the old sidebar entry land on the new page.
      { source: "/my-tasks", destination: "/my-work", permanent: true },
      // Today was folded into My Work — the greeting + stat hero now lead the
      // My Work page, so the standalone /today route redirects there.
      { source: "/today", destination: "/my-work", permanent: true },
    ];
  },
};

// Standalone output is ONLY for the self-hosted Docker image, which opts in via
// BUILD_STANDALONE=1 (see frontend/Dockerfile). It must stay off by default:
// on Vercel, output: "standalone" makes the build produce no serverless output
// and every route returns a platform 404 (X-Vercel-Error: NOT_FOUND).
if (process.env.BUILD_STANDALONE === "1") {
  nextConfig.output = "standalone";
}

export default nextConfig;
