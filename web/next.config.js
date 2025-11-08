/** @type {import('next').NextConfig} */
const nextConfig = {
  // Since this is a client-side app connecting to external API
  output: "standalone",
  // Configure image domains if needed in the future
  images: {
    unoptimized: true,
  },
  // Position dev indicator
  devIndicators: {
    position: "bottom-right",
  },
  // Ensure WebSocket connections work properly
  webpack: (config) => {
    config.resolve.fallback = {
      ...config.resolve.fallback,
      fs: false,
    };
    return config;
  },
};

module.exports = nextConfig;
