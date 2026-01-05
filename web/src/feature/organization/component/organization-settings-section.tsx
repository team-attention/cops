import { Building2, Crown, LogOut, Pencil, User, Users } from 'lucide-react'
import { EditOrganizationDialog } from './edit-organization-dialog'
import { MemberList } from './member-list'
import { LeaveOrganizationDialog } from './leave-organization-dialog'
import { useUserStore } from '@/shared/store/user-store'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/gen/shadcn/ui/card'
import { Badge } from '@/gen/shadcn/ui/badge'
import { Button } from '@/gen/shadcn/ui/button'
import { Separator } from '@/gen/shadcn/ui/separator'

export const OrganizationSettingsSection = () => {
  const { user, organizations, selectedOrganizationId } = useUserStore()

  // Find current organization from organizations array
  const currentOrg = organizations.find(
    (org) => org.id === selectedOrganizationId,
  )

  // If no organization selected, return null
  if (!currentOrg || !user) {
    return null
  }

  // Determine if current user is admin
  const isAdmin = currentOrg.role === 'admin'

  // Calculate isLastOrganization
  const isLastOrganization = organizations.length === 1

  // Calculate isSoleMember (check if org has only one member - the current user)
  // Note: We need to check from the members list, but for now we'll use a simple check
  // This will be properly calculated when we fetch the actual members
  const isSoleMember = false // This will be updated based on actual member count from API

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-3">
          <Building2 className="h-6 w-6" />
          <div className="flex-1">
            <CardTitle>Organization Settings</CardTitle>
            <CardDescription>
              Manage your organization and members
            </CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Organization Info Section */}
        <div className="space-y-4">
          <div className="flex items-start justify-between">
            <div className="space-y-1">
              <h3 className="text-lg font-semibold">{currentOrg.name}</h3>
              <p className="text-sm text-zinc-500">
                Slug: <code className="text-zinc-400">{currentOrg.slug}</code>
              </p>
              <div className="flex items-center gap-2 mt-2">
                <Badge variant={isAdmin ? 'default' : 'secondary'}>
                  {isAdmin ? (
                    <>
                      <Crown className="mr-1 h-3 w-3" />
                      Admin
                    </>
                  ) : (
                    <>
                      <User className="mr-1 h-3 w-3" />
                      Member
                    </>
                  )}
                </Badge>
              </div>
            </div>
            {isAdmin && (
              <EditOrganizationDialog
                organizationId={currentOrg.id}
                currentName={currentOrg.name}
                currentSlug={currentOrg.slug}
                trigger={
                  <Button variant="outline" size="sm">
                    <Pencil className="mr-2 h-4 w-4" />
                    Edit
                  </Button>
                }
              />
            )}
          </div>
        </div>

        <Separator />

        {/* Members Section */}
        <div className="space-y-4">
          <div className="flex items-center gap-2">
            <Users className="h-5 w-5" />
            <h3 className="text-lg font-semibold">Members</h3>
          </div>
          <MemberList
            organizationId={currentOrg.id}
            isAdmin={isAdmin}
            currentUserId={user.id}
          />
        </div>

        <Separator />

        {/* Leave Organization Section */}
        <div className="space-y-4">
          <div className="space-y-2">
            <h3 className="text-lg font-semibold text-red-400">
              Leave Organization
            </h3>
            <p className="text-sm text-zinc-500">
              {isLastOrganization && isSoleMember
                ? 'Leaving will permanently delete this organization and all associated data.'
                : 'You will lose access to all projects and data in this organization.'}
            </p>
          </div>
          <LeaveOrganizationDialog
            organizationId={currentOrg.id}
            organizationName={currentOrg.name}
            isLastOrganization={isLastOrganization}
            isSoleMember={isSoleMember}
            trigger={
              <Button variant="destructive" size="sm">
                <LogOut className="mr-2 h-4 w-4" />
                Leave Organization
              </Button>
            }
          />
        </div>
      </CardContent>
    </Card>
  )
}
