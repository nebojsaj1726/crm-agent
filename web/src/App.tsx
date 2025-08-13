import React, { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import * as Separator from "@radix-ui/react-separator";

interface ApiResponse {
  top_lead: {
    score: number;
    lead_text: string;
  };
  lead_score: string;
  prospect_email: string;
}

async function fetchLead(query: string): Promise<ApiResponse> {
  const res = await fetch("http://localhost:8080/query", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ query }),
  });
  if (!res.ok) {
    throw new Error("API response was not ok");
  }
  return res.json();
}

export default function App() {
  const [query, setQuery] = useState("");
  const [enabled, setEnabled] = useState(false);

  const { data, error, isLoading, refetch, isFetched } = useQuery<ApiResponse>({
    queryKey: ["lead", query],
    queryFn: () => fetchLead(query),
    enabled,
    retry: false,
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!query.trim()) return;
    setEnabled(true);
    refetch();
  }

  useEffect(() => {
    if (isFetched) {
      setEnabled(false);
    }
  }, [isFetched]);

  return (
    <div className="min-h-screen bg-gray-900 text-gray-100 p-6">
      <h1 className="text-4xl font-bold text-center mb-8">CRM Lead Agent</h1>
      <form
        onSubmit={handleSubmit}
        className="flex flex-col sm:flex-row gap-3 max-w-2xl mx-auto"
      >
        <input
          type="text"
          placeholder="Enter a fuzzy lead description..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          className="flex-grow rounded-md px-4 py-2 bg-gray-800 border border-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
          disabled={isLoading}
        />
        <button
          type="submit"
          disabled={isLoading}
          className="bg-blue-600 hover:bg-blue-700 text-white px-6 py-2 rounded-md transition disabled:bg-gray-600"
        >
          Search
        </button>
      </form>

      {isLoading && (
        <div className="flex justify-center mt-6">
          <div className="animate-spin rounded-full h-12 w-12 border-t-4 border-blue-500 border-solid border-gray-700"></div>
        </div>
      )}

      {error && (
        <div className="text-red-600 font-semibold text-center mt-6">
          Error fetching lead: {(error as Error).message}
        </div>
      )}

      {data && (
        <div className="mt-8 space-y-6 max-w-3xl mx-auto">
          <div className="bg-gray-800 p-4 rounded-lg shadow">
            <h2 className="text-xl font-semibold text-blue-400">
              Top Lead (Score: {data.top_lead.score.toFixed(2)})
            </h2>
            <Separator.Root className="bg-gray-700 h-px my-2" />
            <pre className="whitespace-pre-wrap text-gray-300">
              {data.top_lead.lead_text}
            </pre>
          </div>

          <div className="bg-gray-800 p-4 rounded-lg shadow">
            <h2 className="text-xl font-semibold text-green-400">
              Lead Score & Justification
            </h2>
            <Separator.Root className="bg-gray-700 h-px my-2" />
            <pre className="whitespace-pre-wrap text-gray-300">
              {data.lead_score}
            </pre>
          </div>

          <div className="bg-gray-800 p-4 rounded-lg shadow">
            <h2 className="text-xl font-semibold text-purple-400">
              Suggested Prospecting Email
            </h2>
            <Separator.Root className="bg-gray-700 h-px my-2" />
            <pre className="whitespace-pre-wrap text-gray-300">
              {data.prospect_email}
            </pre>
          </div>
        </div>
      )}
    </div>
  );
}
