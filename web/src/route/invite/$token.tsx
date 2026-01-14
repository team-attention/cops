import { useCallback, useState } from 'react'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { Building2, CheckCircle2, Loader2, LogIn, XCircle } from 'lucide-react'
import { Code } from '@connectrpc/connect'
import { useGetInvitationByToken } from '@/feature/organization/hook/use-get-invitation-by-token'
import { useAcceptInvitation } from '@/feature/organization/hook/use-accept-invitation'
import { useUserStore } from '@/shared/store/user-store'
import { Button } from '@/gen/shadcn/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/gen/shadcn/ui/card'
import { Alert, AlertDescription } from '@/gen/shadcn/ui/alert'

// InvitePageState represents the page's internal state.
type InvitePageState =
  | { status: 'idle' }
  | { status: 'accepting' }
  | { status: 'success'; organizationId: string }
  | { status: 'error'; message: string }

export const Route = createFileRoute('/invite/$token')({
  component: InviteAcceptPage,
})

function InviteAcceptPage() {
  const { token } = Route.useParams()
  const navigate = useNavigate()
  const { user, addOrganization, setSelectedOrganizationId } = useUserStore()

  const [state, setState] = useState<InvitePageState>({ status: 'idle' })

  const {
    data: invitationData,
    isLoading: isLoadingInvitation,
    error: invitationError,
  } = useGetInvitationByToken({ token })

  const acceptMutation = useAcceptInvitation()

  const handleAccept = useCallback(async () => {
    setState({ status: 'accepting' })
    try {
      const result = await acceptMutation.mutateAsync({ token })

      if (result.success && result.organizationId) {
        // Add the new organization to the store
        addOrganization({
          id: result.organizationId,
          name: invitationData?.organizationName || 'Organization',
          slug: '', // Will be fetched when user accesses the org
          role: 'member',
        })

        // Select the new organization
        setSelectedOrganizationId(result.organizationId)

        setState({ status: 'success', organizationId: result.organizationId })

        // Navigate to dashboard after short delay
        setTimeout(() => {
          navigate({ to: '/dashboard' })
        }, 2000)
      } else {
        setState({ status: 'error', message: 'Failed to accept invitation' })
      }
    } catch (error) {
      const connectError = error as { code?: Code; message?: string }
      let errorMessage = 'Failed to accept invitation'

      if (connectError.code === Code.Unauthenticated) {
        // Redirect to login with return URL
        navigate({
          to: '/auth',
          search: { redirect: `/invite/${token}` },
        })
        return
      } else if (connectError.code === Code.NotFound) {
        errorMessage = 'Invitation not found or has expired'
      } else if (connectError.code === Code.FailedPrecondition) {
        if (connectError.message?.includes('email mismatch')) {
          errorMessage =
            'Your email address does not match the invitation. Please log in with the correct account.'
        } else {
          errorMessage = 'This invitation is no longer valid'
        }
      } else if (connectError.message) {
        errorMessage = connectError.message
      }

      setState({ status: 'error', message: errorMessage })
    }
  }, [
    acceptMutation,
    token,
    invitationData?.organizationName,
    addOrganization,
    setSelectedOrganizationId,
    navigate,
  ])

  const handleLogin = useCallback(() => {
    navigate({
      to: '/auth',
      search: { redirect: `/invite/${token}` },
    })
  }, [navigate, token])

  // Loading state
  if (isLoadingInvitation) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-zinc-950">
        <Card className="w-full max-w-md border-zinc-800 bg-zinc-900/50">
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-violet-400" />
            <p className="mt-4 text-zinc-400">Loading invitation...</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  // Error state - invitation not found or invalid
  if (invitationError || !invitationData) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-zinc-950">
        <Card className="w-full max-w-md border-zinc-800 bg-zinc-900/50">
          <CardHeader className="text-center">
            <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-red-500/20">
              <XCircle className="h-8 w-8 text-red-400" />
            </div>
            <CardTitle>Invalid Invitation</CardTitle>
            <CardDescription>
              This invitation link is invalid, expired, or has already been used.
            </CardDescription>
          </CardHeader>
          <CardFooter className="justify-center">
            <Button variant="outline" onClick={() => navigate({ to: '/' })}>
              Go to Home
            </Button>
          </CardFooter>
        </Card>
      </div>
    )
  }

  // Success state
  if (state.status === 'success') {
    return (
      <div className="flex min-h-screen items-center justify-center bg-zinc-950">
        <Card className="w-full max-w-md border-zinc-800 bg-zinc-900/50">
          <CardHeader className="text-center">
            <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-green-500/20">
              <CheckCircle2 className="h-8 w-8 text-green-400" />
            </div>
            <CardTitle>Welcome!</CardTitle>
            <CardDescription>
              You have successfully joined{' '}
              <strong>{invitationData.organizationName}</strong>. Redirecting to
              dashboard...
            </CardDescription>
          </CardHeader>
          <CardContent className="flex justify-center">
            <Loader2 className="h-5 w-5 animate-spin text-zinc-400" />
          </CardContent>
        </Card>
      </div>
    )
  }

  // Main invitation view
  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-950">
      <Card className="w-full max-w-md border-zinc-800 bg-zinc-900/50">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-violet-500/20">
            <Building2 className="h-8 w-8 text-violet-400" />
          </div>
          <CardTitle>You're Invited!</CardTitle>
          <CardDescription>
            You've been invited to join{' '}
            <strong className="text-zinc-200">
              {invitationData.organizationName}
            </strong>{' '}
            on C-Ops.
          </CardDescription>
        </CardHeader>

        <CardContent className="space-y-4">
          <div className="rounded-lg border border-zinc-800 bg-zinc-950/50 p-4">
            <p className="text-sm text-zinc-400">
              Invitation sent to:{' '}
              <span className="font-medium text-zinc-200">
                {invitationData.invitation?.email}
              </span>
            </p>
          </div>

          {state.status === 'error' && (
            <Alert className="border-red-900/50 bg-red-950/20">
              <AlertDescription className="text-sm text-red-200">
                {state.message}
              </AlertDescription>
            </Alert>
          )}

          {!user && (
            <Alert className="border-amber-900/50 bg-amber-950/20">
              <AlertDescription className="text-sm text-amber-200">
                Please log in to accept this invitation. Make sure to use the
                same email address the invitation was sent to.
              </AlertDescription>
            </Alert>
          )}
        </CardContent>

        <CardFooter className="flex justify-center gap-3">
          {user ? (
            <Button
              onClick={handleAccept}
              disabled={state.status === 'accepting'}
              className="w-full"
            >
              {state.status === 'accepting' ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <CheckCircle2 className="mr-2 h-4 w-4" />
              )}
              Accept Invitation
            </Button>
          ) : (
            <Button onClick={handleLogin} className="w-full">
              <LogIn className="mr-2 h-4 w-4" />
              Log In to Accept
            </Button>
          )}
        </CardFooter>
      </Card>
    </div>
  )
}
