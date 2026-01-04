import { useState, useCallback } from 'react'
import { Loader2, Shield, User, MoreHorizontal, UserMinus, ShieldCheck, ShieldOff } from 'lucide-react'
import { Code } from '@connectrpc/connect'
import { useGetOrganizationMembers } from '../hook/use-get-organization-members'
import { useUpdateMemberRole } from '../hook/use-update-member-role'
import { useRemoveMember } from '../hook/use-remove-member'
import { Avatar, AvatarFallback, AvatarImage } from '@/gen/shadcn/ui/avatar'
import { Button } from '@/gen/shadcn/ui/button'
import { Badge } from '@/gen/shadcn/ui/badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/gen/shadcn/ui/dropdown-menu'
import { Skeleton } from '@/gen/shadcn/ui/skeleton'
import { Alert, AlertDescription } from '@/gen/shadcn/ui/alert'

import type { MemberWithDetails } from '../type/member'

interface MemberListProps {
  // organizationId is the ID of the organization
  organizationId: string
  // isAdmin indicates if the current user has admin role
  isAdmin: boolean
  // currentUserId is the ID of the current user
  currentUserId: string
}

type ConfirmDialogState =
  | { type: 'remove'; member: MemberWithDetails }
  | { type: 'changeRole'; member: MemberWithDetails; newRole: string }
  | null

export const MemberList = ({ organizationId, isAdmin, currentUserId }: MemberListProps) => {
  const [confirmDialog, setConfirmDialog] = useState<ConfirmDialogState>(null)
  const [actionLoading, setActionLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const membersQuery = useGetOrganizationMembers({ organizationId })
  const updateRoleMutation = useUpdateMemberRole()
  const removeMemberMutation = useRemoveMember()

  const handleRoleChange = useCallback(async () => {
    if (!confirmDialog || confirmDialog.type !== 'changeRole') return

    setActionLoading(true)
    setError(null)
    try {
      await updateRoleMutation.mutateAsync({
        organizationId,
        userId: confirmDialog.member.userId,
        role: confirmDialog.newRole,
      })
      await membersQuery.refetch()
      setConfirmDialog(null)
    } catch (error) {
      const connectError = error as { code?: Code; message?: string }
      let errorMessage = 'Failed to update role'

      if (connectError.code === Code.PermissionDenied) {
        errorMessage = 'You do not have permission to change roles'
      } else if (connectError.code === Code.FailedPrecondition) {
        errorMessage = 'Cannot demote the last admin'
      } else if (connectError.message) {
        errorMessage = connectError.message
      }

      setError(errorMessage)
    } finally {
      setActionLoading(false)
    }
  }, [confirmDialog, organizationId, updateRoleMutation, membersQuery])

  const handleRemoveMember = useCallback(async () => {
    if (!confirmDialog || confirmDialog.type !== 'remove') return

    setActionLoading(true)
    setError(null)
    try {
      await removeMemberMutation.mutateAsync({
        organizationId,
        userId: confirmDialog.member.userId,
      })
      await membersQuery.refetch()
      setConfirmDialog(null)
    } catch (error) {
      const connectError = error as { code?: Code; message?: string }
      let errorMessage = 'Failed to remove member'

      if (connectError.code === Code.PermissionDenied) {
        errorMessage = 'You do not have permission to remove members'
      } else if (connectError.code === Code.FailedPrecondition) {
        errorMessage = 'Cannot remove the last admin'
      } else if (connectError.message) {
        errorMessage = connectError.message
      }

      setError(errorMessage)
    } finally {
      setActionLoading(false)
    }
  }, [confirmDialog, organizationId, removeMemberMutation, membersQuery])

  if (membersQuery.isLoading) {
    return (
      <div className="space-y-4">
        {[1, 2, 3].map((i) => (
          <div key={i} className="flex items-center gap-4">
            <Skeleton className="h-10 w-10 rounded-full" />
            <div className="flex-1 space-y-2">
              <Skeleton className="h-4 w-48" />
              <Skeleton className="h-3 w-32" />
            </div>
          </div>
        ))}
      </div>
    )
  }

  if (membersQuery.isError) {
    return (
      <Alert className="border-red-900/50 bg-red-950/20">
        <AlertDescription className="text-sm text-red-200">
          Failed to load members. Please try again.
        </AlertDescription>
      </Alert>
    )
  }

  const members: MemberWithDetails[] = (membersQuery.data?.members || []).map((m) => ({
    userId: m.userId,
    email: m.email,
    name: m.name,
    avatarUrl: m.avatarUrl,
    role: m.role as 'admin' | 'member',
  }))
  const adminCount = members.filter((m) => m.role === 'admin').length

  return (
    <>
      <div className="space-y-4">
        {error && (
          <Alert className="border-red-900/50 bg-red-950/20">
            <AlertDescription className="text-sm text-red-200">
              {error}
            </AlertDescription>
          </Alert>
        )}

        {members.map((member) => {
          const isCurrentUser = member.userId === currentUserId
          const isSoleAdmin = member.role === 'admin' && adminCount === 1

          return (
            <div key={member.userId} className="flex items-center gap-4">
              <Avatar>
                <AvatarImage src={member.avatarUrl} />
                <AvatarFallback>
                  {member.name
                    .split(' ')
                    .map((n) => n[0])
                    .join('')
                    .toUpperCase()}
                </AvatarFallback>
              </Avatar>

              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <p className="font-medium text-sm">
                    {member.name}
                    {isCurrentUser && (
                      <span className="ml-2 text-xs text-zinc-500">(You)</span>
                    )}
                  </p>
                  <Badge variant={member.role === 'admin' ? 'default' : 'secondary'}>
                    {member.role === 'admin' ? (
                      <>
                        <Shield className="mr-1 h-3 w-3" />
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
                <p className="text-xs text-zinc-500">{member.email}</p>
              </div>

              {isAdmin && (
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" size="sm">
                      <MoreHorizontal className="h-4 w-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    {member.role === 'admin' ? (
                      <DropdownMenuItem
                        onClick={() =>
                          setConfirmDialog({
                            type: 'changeRole',
                            member,
                            newRole: 'member',
                          })
                        }
                        disabled={isSoleAdmin}
                      >
                        <ShieldOff className="mr-2 h-4 w-4" />
                        Demote to Member
                      </DropdownMenuItem>
                    ) : (
                      <DropdownMenuItem
                        onClick={() =>
                          setConfirmDialog({
                            type: 'changeRole',
                            member,
                            newRole: 'admin',
                          })
                        }
                      >
                        <ShieldCheck className="mr-2 h-4 w-4" />
                        Promote to Admin
                      </DropdownMenuItem>
                    )}
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      onClick={() =>
                        setConfirmDialog({
                          type: 'remove',
                          member,
                        })
                      }
                      disabled={isSoleAdmin}
                      className="text-red-400"
                    >
                      <UserMinus className="mr-2 h-4 w-4" />
                      Remove Member
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              )}
            </div>
          )
        })}
      </div>

      {/* Confirmation Dialog */}
      {confirmDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80">
          <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-6 max-w-md w-full">
            <h3 className="text-lg font-semibold mb-2">
              {confirmDialog.type === 'remove'
                ? 'Remove Member'
                : confirmDialog.type === 'changeRole'
                  ? 'Change Role'
                  : ''}
            </h3>
            <p className="text-sm text-zinc-400 mb-4">
              {confirmDialog.type === 'remove' && (
                <>
                  Are you sure you want to remove <strong>{confirmDialog.member.name}</strong> from
                  this organization? They will lose access to all projects and data.
                </>
              )}
              {confirmDialog.type === 'changeRole' && (
                <>
                  Are you sure you want to change <strong>{confirmDialog.member.name}</strong>'s role
                  to <strong>{confirmDialog.newRole}</strong>?
                </>
              )}
            </p>
            <div className="flex gap-2 justify-end">
              <Button variant="outline" onClick={() => setConfirmDialog(null)} disabled={actionLoading}>
                Cancel
              </Button>
              <Button
                onClick={() => {
                  if (confirmDialog.type === 'remove') {
                    handleRemoveMember()
                  } else if (confirmDialog.type === 'changeRole') {
                    handleRoleChange()
                  }
                }}
                disabled={actionLoading}
              >
                {actionLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Confirm
              </Button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
