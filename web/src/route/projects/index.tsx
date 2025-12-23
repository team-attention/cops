import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/projects/')({
  component: ProjectsListPage,
})

function ProjectsListPage() {
  return (
    <div className="min-h-screen bg-zinc-950 p-8">
      <h1 className="text-2xl font-bold text-zinc-100">All Projects</h1>
      <p className="mt-2 text-zinc-500">Projects list page (coming soon)</p>
    </div>
  )
}
