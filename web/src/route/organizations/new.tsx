import { createFileRoute } from '@tanstack/react-router'
import { OrganizationForm } from '@/feature/organization/component/organization-form'

export const Route = createFileRoute('/organizations/new')({
  component: OrganizationNewPage,
})

// OrganizationNewPage displays the organization creation form.
function OrganizationNewPage() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-950">
      <div className="w-full max-w-md px-4">
        <OrganizationForm />
      </div>
    </div>
  )
}
