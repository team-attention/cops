import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/sessions/')({
  component: SessionsListPage,
})

function SessionsListPage() {
  return (
    <div className="min-h-screen bg-zinc-950 p-8">
      <h1 className="text-2xl font-bold text-zinc-100">All Sessions</h1>
      <p className="mt-2 text-zinc-500">Sessions list page (coming soon)</p>
    </div>
  )
}
