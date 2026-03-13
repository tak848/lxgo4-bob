"use client";

import { use, useCallback, useEffect, useState } from "react";
import Link from "next/link";

import { Skeleton } from "@/components/ui/skeleton";

import { MarkdownContent } from "./markdown-content";

export default function DocPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = use(params);
  const [content, setContent] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchDoc = useCallback(async () => {
    const res = await fetch(`/api/docs/${slug}`);
    if (res.ok) {
      const data = (await res.json()) as { content: string };
      setContent(data.content);
    }
    setLoading(false);
  }, [slug]);

  useEffect(() => {
    void fetchDoc();
  }, [fetchDoc]);

  if (loading) {
    return (
      <div className="mx-auto max-w-3xl space-y-4 p-8">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-3/4" />
        <Skeleton className="h-32 w-full" />
      </div>
    );
  }

  if (!content) {
    return (
      <div className="mx-auto max-w-3xl p-8">
        <p className="text-muted-foreground">Document not found.</p>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-3xl p-8">
      <div className="mb-6">
        <Link
          href="/docs"
          className="text-muted-foreground hover:text-foreground text-sm"
        >
          &larr; Docs Index
        </Link>
      </div>
      <MarkdownContent content={content} />
    </div>
  );
}
