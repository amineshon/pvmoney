import type { NextConfig } from "next";

const apiUrl = process.env.API_URL || "http://localhost:8080";

const nextConfig: NextConfig = {
  output: "standalone",
  eslint: { ignoreDuringBuilds: true },
  async rewrites() {
    return [
      { source: "/api/:path*", destination: `${apiUrl}/api/:path*` },
      { source: "/health", destination: `${apiUrl}/health` },
    ];
  },
};

export default nextConfig;
