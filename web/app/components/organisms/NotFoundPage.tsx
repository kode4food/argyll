import React from "react";
import { AlertTriangle } from "lucide-react";
import Link from "next/link";

const NotFoundPage: React.FC = () => {
  return (
    <div className="bg-neutral-bg flex h-screen items-center justify-center">
      <div className="text-center">
        <AlertTriangle className="text-collector-text mx-auto mb-4 h-16 w-16" />
        <h1 className="text-neutral-text mb-2 text-3xl font-bold">
          404 - Page Not Found
        </h1>
        <p className="text-neutral-text mb-6 max-w-md">
          The page you&apos;re looking for doesn&apos;t exist. Check the URL or
          return to the overview.
        </p>
        <Link
          href="/"
          className="bg-info hover:bg-processor-text rounded px-6 py-3 font-medium text-white"
        >
          Back to Overview
        </Link>
      </div>
    </div>
  );
};

export default NotFoundPage;
