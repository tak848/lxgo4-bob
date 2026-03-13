"use client";

import dynamic from "next/dynamic";
import remarkGfm from "remark-gfm";

const ReactMarkdown = dynamic(() => import("react-markdown"), { ssr: false });

export function MarkdownContent({ content }: { content: string }) {
  return (
    <article className="max-w-none space-y-4">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        children={content}
        components={{
          h1: ({ children }) => (
            <h1 className="mt-0 mb-6 text-3xl font-bold">{children}</h1>
          ),
          h2: ({ children }) => (
            <h2 className="mt-10 mb-4 border-b pb-2 text-2xl font-semibold">
              {children}
            </h2>
          ),
          h3: ({ children }) => (
            <h3 className="mt-8 mb-3 text-xl font-semibold">{children}</h3>
          ),
          p: ({ children }) => <p className="my-3 leading-7">{children}</p>,
          code: ({ className, children }) => {
            const isBlock = className?.startsWith("language-");
            if (isBlock) {
              return (
                <code
                  className={`${className ?? ""} block overflow-x-auto rounded-lg bg-neutral-900 p-4 text-sm text-neutral-100`}
                >
                  {children}
                </code>
              );
            }
            return (
              <code className="rounded bg-neutral-200 px-1.5 py-0.5 text-sm dark:bg-neutral-800">
                {children}
              </code>
            );
          },
          pre: ({ children }) => <pre className="my-4">{children}</pre>,
          table: ({ children }) => (
            <div className="my-4 overflow-x-auto">
              <table className="min-w-full border-collapse border border-neutral-300 text-sm dark:border-neutral-700">
                {children}
              </table>
            </div>
          ),
          th: ({ children }) => (
            <th className="border border-neutral-300 bg-neutral-100 px-3 py-2 text-left font-semibold dark:border-neutral-700 dark:bg-neutral-800">
              {children}
            </th>
          ),
          td: ({ children }) => (
            <td className="border border-neutral-300 px-3 py-2 dark:border-neutral-700">
              {children}
            </td>
          ),
          a: ({ href, children }) => (
            <a
              href={href}
              className="text-blue-600 underline hover:text-blue-800 dark:text-blue-400"
            >
              {children}
            </a>
          ),
          blockquote: ({ children }) => (
            <blockquote className="border-l-4 border-neutral-300 pl-4 italic text-neutral-600 dark:border-neutral-600 dark:text-neutral-400">
              {children}
            </blockquote>
          ),
          ul: ({ children }) => (
            <ul className="my-2 list-disc space-y-1 pl-6">{children}</ul>
          ),
          ol: ({ children }) => (
            <ol className="my-2 list-decimal space-y-1 pl-6">{children}</ol>
          ),
          li: ({ children }) => <li className="leading-7">{children}</li>,
          hr: () => (
            <hr className="my-8 border-neutral-300 dark:border-neutral-700" />
          ),
          strong: ({ children }) => (
            <strong className="font-semibold">{children}</strong>
          ),
        }}
      />
    </article>
  );
}
